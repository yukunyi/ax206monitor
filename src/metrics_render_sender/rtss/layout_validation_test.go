package rtss

import "testing"

func TestSharedMemoryLayoutValidAcceptsReadableArray(t *testing.T) {
	valid := sharedMemoryLayoutValid(64*1024, 36, 4096, 1024, 16, 32)
	if !valid {
		t.Fatalf("expected shared memory layout to be valid")
	}
}

func TestSharedMemoryLayoutValidRejectsRegionOverflow(t *testing.T) {
	valid := sharedMemoryLayoutValid(4096, 36, 2048, 512, 8, 32)
	if valid {
		t.Fatalf("expected overflowed shared memory layout to be invalid")
	}
}

func TestSharedMemoryLayoutValidRejectsTooManyEntries(t *testing.T) {
	valid := sharedMemoryLayoutValid(64*1024, 36, 4096, 64, maxSharedMemoryAppEntries+1, 32)
	if valid {
		t.Fatalf("expected layout with too many entries to be invalid")
	}
}

func TestSharedMemoryLayoutValidRejectsSmallRegion(t *testing.T) {
	valid := sharedMemoryLayoutValid(8, 36, 4, 32, 1, 32)
	if valid {
		t.Fatalf("expected undersized region to be invalid")
	}
}
