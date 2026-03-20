package output

import "testing"

func TestBuildManagerUsesDedicatedAX206Handler(t *testing.T) {
	manager, configs := BuildManager([]OutputConfig{{Type: TypeAX206USB}}, false)
	if manager == nil {
		t.Fatal("expected manager")
	}
	defer manager.Close()

	if len(configs) != 1 || configs[0].Type != TypeAX206USB {
		t.Fatalf("unexpected configs: %#v", configs)
	}
	if len(manager.handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(manager.handlers))
	}
	if _, ok := manager.handlers[0].(*AX206USBOutputHandler); !ok {
		t.Fatalf("expected dedicated AX206 handler, got %T", manager.handlers[0])
	}
}

func TestResolveConfigsAddsMemImgOnlyWhenForced(t *testing.T) {
	configs := []OutputConfig{{Type: TypeAX206USB}}

	normal := ResolveConfigs(configs, false)
	if len(normal) != 1 || normal[0].Type != TypeAX206USB {
		t.Fatalf("unexpected normal configs: %#v", normal)
	}

	preview := ResolveConfigs(configs, true)
	if len(preview) != 2 {
		t.Fatalf("expected 2 preview configs, got %#v", preview)
	}
	if preview[0].Type != TypeAX206USB || preview[1].Type != TypeMemImg {
		t.Fatalf("unexpected preview configs: %#v", preview)
	}
}

func TestBuildManagerIgnoresDisabledOutputs(t *testing.T) {
	enabled := false
	manager, configs := BuildManager([]OutputConfig{{
		Type:        TypeAX206USB,
		Enabled:     &enabled,
		ReconnectMS: 1200,
	}}, false)
	if manager == nil {
		t.Fatal("expected manager")
	}
	defer manager.Close()

	if len(configs) != 0 {
		t.Fatalf("expected no active configs, got %#v", configs)
	}
	if len(manager.handlers) != 0 {
		t.Fatalf("expected no handlers, got %d", len(manager.handlers))
	}
}

func TestNormalizeConfigsPreservesHTTPPushProtocolFields(t *testing.T) {
	configs := NormalizeConfigs([]OutputConfig{{
		Type:        TypeHTTPPush,
		URL:         " http://127.0.0.1/push ",
		Method:      " put ",
		BodyMode:    "formdata",
		Format:      "baseline_jpeg",
		Quality:     91,
		ContentType: " image/jpeg ",
		Headers: []HTTPKeyValue{
			{Key: " Authorization ", Value: " Bearer demo "},
			{Key: "X-Test", Value: " ok "},
			{Key: " ", Value: "skip"},
		},
		AuthType:     " bearer ",
		AuthUsername: " demo ",
		AuthPassword: " secret ",
		AuthToken:    " token ",
		TimeoutMS:    42,
		FileField:    " upload ",
		FileName:     " frame.jpg ",
		FormFields: []HTTPKeyValue{
			{Key: "device", Value: " ax206 "},
		},
		SuccessCodes: []int{204, 201, 204, 999},
	}})

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %#v", configs)
	}
	cfg := configs[0]
	if cfg.Method != "PUT" {
		t.Fatalf("unexpected method: %#v", cfg)
	}
	if cfg.BodyMode != "multipart" {
		t.Fatalf("unexpected body mode: %#v", cfg)
	}
	if cfg.Format != "jpeg_baseline" {
		t.Fatalf("unexpected format: %#v", cfg)
	}
	if cfg.TimeoutMS != 100 {
		t.Fatalf("unexpected timeout: %#v", cfg)
	}
	if cfg.ContentType != "image/jpeg" {
		t.Fatalf("unexpected content type: %#v", cfg)
	}
	if cfg.AuthType != "bearer" || cfg.AuthToken != "token" {
		t.Fatalf("unexpected auth: %#v", cfg)
	}
	if cfg.FileField != "upload" || cfg.FileName != "frame.jpg" {
		t.Fatalf("unexpected multipart names: %#v", cfg)
	}
	if len(cfg.Headers) != 2 || cfg.Headers[0].Key != "Authorization" || cfg.Headers[0].Value != "Bearer demo" {
		t.Fatalf("unexpected headers: %#v", cfg.Headers)
	}
	if len(cfg.FormFields) != 1 || cfg.FormFields[0].Key != "device" || cfg.FormFields[0].Value != "ax206" {
		t.Fatalf("unexpected form fields: %#v", cfg.FormFields)
	}
	if len(cfg.SuccessCodes) != 2 || cfg.SuccessCodes[0] != 201 || cfg.SuccessCodes[1] != 204 {
		t.Fatalf("unexpected success codes: %#v", cfg.SuccessCodes)
	}
}

func TestNormalizeConfigsPreservesTCPPushBinaryFields(t *testing.T) {
	configs := NormalizeConfigs([]OutputConfig{{
		Type:    TypeTCPPush,
		URL:     " tcp://127.0.0.1:9100 ",
		Format:  "rgb565_rle",
		Quality: 88,
		Headers: []HTTPKeyValue{
			{Key: " X-Test ", Value: " ok "},
		},
		UploadToken:    " token ",
		TimeoutMS:      80,
		IdleTimeoutSec: 2,
		BusyCheckMS:    20,
		FileName:       " frame.jpg ",
		SuccessCodes:   []int{202, 200, 202},
	}})

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %#v", configs)
	}
	cfg := configs[0]
	if cfg.URL != "tcp://127.0.0.1:9100" {
		t.Fatalf("unexpected url: %#v", cfg)
	}
	if cfg.Format != "rgb565le_rle" {
		t.Fatalf("unexpected format: %#v", cfg)
	}
	if cfg.TimeoutMS != 100 {
		t.Fatalf("unexpected timeout: %#v", cfg)
	}
	if cfg.IdleTimeoutSec != 5 {
		t.Fatalf("unexpected idle timeout: %#v", cfg)
	}
	if cfg.BusyCheckMS != 100 {
		t.Fatalf("unexpected busy check: %#v", cfg)
	}
	if cfg.UploadToken != "token" {
		t.Fatalf("unexpected upload token: %#v", cfg)
	}
	if cfg.FileName != "frame.jpg" {
		t.Fatalf("unexpected file name: %#v", cfg)
	}
	if len(cfg.Headers) != 0 {
		t.Fatalf("expected tcp headers ignored, got %#v", cfg.Headers)
	}
	if len(cfg.SuccessCodes) != 2 || cfg.SuccessCodes[0] != 200 || cfg.SuccessCodes[1] != 202 {
		t.Fatalf("unexpected success codes: %#v", cfg.SuccessCodes)
	}
}

func TestBuildManagerKeepsTCPPushWithoutToken(t *testing.T) {
	manager, configs := BuildManager([]OutputConfig{{
		Type:    TypeTCPPush,
		Enabled: cloneEnabledValue(true),
		URL:     "tcp://127.0.0.1:9100",
	}}, false)
	if manager == nil {
		t.Fatal("expected manager")
	}
	defer manager.Close()

	if len(configs) != 1 || configs[0].Type != TypeTCPPush {
		t.Fatalf("expected tcp push config preserved, got %#v", configs)
	}
	if len(manager.handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(manager.handlers))
	}
}
