package rtsssource

import "testing"

func TestRTSSClient_ReturnsZeroWhenDisconnected(t *testing.T) {
	client := &RTSSClient{}

	cases := []struct {
		name     string
		wantUnit string
	}{
		{name: "rtss_connected", wantUnit: ""},
		{name: "rtss_fps", wantUnit: "FPS"},
		{name: "rtss_frametime_ms", wantUnit: "ms"},
		{name: "rtss_fps_avg", wantUnit: "FPS"},
		{name: "rtss_fps_1p_low", wantUnit: "FPS"},
		{name: "rtss_fps_01p_low", wantUnit: "FPS"},
		{name: "rtss_frametime_min_ms", wantUnit: "ms"},
		{name: "rtss_frametime_avg_ms", wantUnit: "ms"},
		{name: "rtss_frametime_max_ms", wantUnit: "ms"},
		{name: "rtss_frametime_p99_ms", wantUnit: "ms"},
		{name: "rtss_frametime_p999_ms", wantUnit: "ms"},
		{name: "rtss_max_fps", wantUnit: "FPS"},
		{name: "rtss_active_apps", wantUnit: ""},
		{name: "rtss_foreground_pid", wantUnit: ""},
	}

	for _, tc := range cases {
		value, unit, ok, err := client.GetMonitorValueByNameCached(tc.name)
		if err != nil {
			t.Fatalf("GetMonitorValueByNameCached(%q) err=%v", tc.name, err)
		}
		if !ok {
			t.Fatalf("GetMonitorValueByNameCached(%q) ok=false, want true", tc.name)
		}
		if unit != tc.wantUnit {
			t.Fatalf("GetMonitorValueByNameCached(%q) unit=%q, want %q", tc.name, unit, tc.wantUnit)
		}
		if value != 0 {
			t.Fatalf("GetMonitorValueByNameCached(%q) value=%v, want 0", tc.name, value)
		}
	}
}

func TestRTSSClient_UnknownMonitor(t *testing.T) {
	client := &RTSSClient{}

	_, _, ok, err := client.GetMonitorValueByNameCached("rtss_not_exists")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("ok=true for unknown monitor")
	}
}
