package main

import "testing"

func TestNewRenderManagerWithHistoryReusesExistingStore(t *testing.T) {
	history := newRenderHistoryStore()
	manager := NewRenderManagerWithHistory(nil, nil, history)
	if manager == nil {
		t.Fatalf("expected render manager")
	}
	if manager.history != history {
		t.Fatalf("expected render manager to reuse provided history store")
	}
}

func TestNewRenderManagerWithHistoryCreatesStoreWhenNil(t *testing.T) {
	manager := NewRenderManagerWithHistory(nil, nil, nil)
	if manager == nil {
		t.Fatalf("expected render manager")
	}
	if manager.history == nil {
		t.Fatalf("expected render manager to create history store")
	}
}
