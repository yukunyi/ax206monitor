package main

import "testing"

func TestDefaultRTSSCollectorEnabledForPlatformDisabledOnWindows(t *testing.T) {
	if defaultRTSSCollectorEnabledForPlatform("windows") {
		t.Fatalf("expected RTSS collector to be disabled by default on windows")
	}
}

func TestDefaultRTSSCollectorEnabledForPlatformDisabledOnLinux(t *testing.T) {
	if defaultRTSSCollectorEnabledForPlatform("linux") {
		t.Fatalf("expected RTSS collector to be disabled by default on linux")
	}
}
