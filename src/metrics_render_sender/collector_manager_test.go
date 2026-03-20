package main

import (
	"testing"
	"time"
)

type testConfigurableCollector struct {
	*BaseCollector
	getAllItemsCalls int
	applyConfigCalls int
}

func newTestConfigurableCollector(name string) *testConfigurableCollector {
	return &testConfigurableCollector{
		BaseCollector: NewBaseCollector(name),
	}
}

func (c *testConfigurableCollector) GetAllItems() map[string]*CollectItem {
	c.getAllItemsCalls++
	return c.ItemsSnapshot()
}

func (c *testConfigurableCollector) UpdateItems() error {
	return nil
}

func (c *testConfigurableCollector) ApplyConfig(cfg *MonitorConfig) {
	c.applyConfigCalls++
	c.clearItems()
	item := NewCollectItem("test.metric", "Test metric", "", 0, 0, 0)
	c.setItem(item.GetName(), item)
}

func TestCollectorManagerApplyConfigDoesNotDiscover(t *testing.T) {
	manager := NewCollectorManager()
	collector := newTestConfigurableCollector("test.collector")
	manager.RegisterCollector(collector)
	manager.mutex.Lock()
	manager.collectorEnabled[collector.Name()] = true
	manager.mutex.Unlock()

	manager.ApplyConfig(&MonitorConfig{}, []string{"test.metric"})

	if collector.applyConfigCalls != 1 {
		t.Fatalf("expected ApplyConfig to be called once, got %d", collector.applyConfigCalls)
	}
	if collector.getAllItemsCalls != 0 {
		t.Fatalf("expected ApplyConfig to avoid discovery, got %d GetAllItems calls", collector.getAllItemsCalls)
	}
	if item := manager.Get("test.metric"); item == nil {
		t.Fatal("expected snapshot sync to register configured item")
	}
}

func TestRegisterCollectorWithConfigDoesNotApplyConfig(t *testing.T) {
	manager := NewCollectorManager()
	collector := newTestConfigurableCollector("test.collector")

	registerCollectorWithConfig(manager, &MonitorConfig{}, collector, true)

	if collector.applyConfigCalls != 0 {
		t.Fatalf("expected collector registration to be side-effect free, got %d ApplyConfig calls", collector.applyConfigCalls)
	}
}

func TestGetCollectorManagerReturnsExistingManagerWithoutApply(t *testing.T) {
	ResetGlobalCollectorManager()
	defer ResetGlobalCollectorManager()

	manager := NewCollectorManager()
	collector := newTestConfigurableCollector("test.collector")
	manager.RegisterCollector(collector)

	globalCollectorMu.Lock()
	globalCollectorConfig = &MonitorConfig{}
	globalCollectorManager = manager
	globalCollectorMu.Unlock()

	got := GetCollectorManager()
	if got != manager {
		t.Fatal("expected existing global manager to be returned")
	}
	if collector.applyConfigCalls != 0 {
		t.Fatalf("expected GetCollectorManager to be side-effect free, got %d ApplyConfig calls", collector.applyConfigCalls)
	}
}

func TestGoNativeSystemCollectorGetAllItemsDoesNotWaitGlobalConfigLock(t *testing.T) {
	collector := NewGoNativeSystemCollector()

	globalCollectorMu.Lock()
	done := make(chan struct{})
	go func() {
		_ = collector.GetAllItems()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		globalCollectorMu.Unlock()
		t.Fatal("GetAllItems blocked on global collector lock")
	}
	globalCollectorMu.Unlock()
}

func TestSetGlobalCollectorConfigAppliesWithoutDiscover(t *testing.T) {
	ResetGlobalCollectorManager()
	defer ResetGlobalCollectorManager()

	manager := NewCollectorManager()
	collector := newTestConfigurableCollector("test.collector")
	manager.RegisterCollector(collector)
	manager.mutex.Lock()
	manager.collectorEnabled[collector.Name()] = true
	manager.mutex.Unlock()

	globalCollectorMu.Lock()
	globalCollectorManager = manager
	globalCollectorConfig = nil
	globalCollectorMu.Unlock()

	SetGlobalCollectorConfig(&MonitorConfig{})

	if collector.applyConfigCalls != 1 {
		t.Fatalf("expected SetGlobalCollectorConfig to apply once, got %d", collector.applyConfigCalls)
	}
	if collector.getAllItemsCalls != 0 {
		t.Fatalf("expected SetGlobalCollectorConfig to avoid discovery, got %d GetAllItems calls", collector.getAllItemsCalls)
	}
}
