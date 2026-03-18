package main

import "testing"

func TestTrimDiskPartitionSuffix(t *testing.T) {
	cases := map[string]string{
		"nvme0n1":   "nvme0n1",
		"nvme0n1p1": "nvme0n1",
		"mmcblk0":   "mmcblk0",
		"mmcblk0p2": "mmcblk0",
		"sda":       "sda",
		"sda1":      "sda",
		"xvdb3":     "xvdb",
	}
	for input, want := range cases {
		if got := trimDiskPartitionSuffix(input); got != want {
			t.Fatalf("trimDiskPartitionSuffix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsLinuxPseudoDiskName(t *testing.T) {
	cases := map[string]bool{
		"loop0":   true,
		"zram0":   true,
		"ram1":    true,
		"fd0":     true,
		"nvme0n1": false,
		"sda":     false,
	}
	for input, want := range cases {
		if got := isLinuxPseudoDiskName(input); got != want {
			t.Fatalf("isLinuxPseudoDiskName(%q) = %v, want %v", input, got, want)
		}
	}
}
