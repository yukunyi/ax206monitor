package rtss

const maxSharedMemoryAppEntries = 4096

func sharedMemoryLayoutValid(
	regionSize uintptr,
	headerSize uintptr,
	appArrOffset uint32,
	appEntrySize uint32,
	appArrSize uint32,
	minEntrySize uint32,
) bool {
	if regionSize < headerSize {
		return false
	}
	if appEntrySize < minEntrySize || appArrSize == 0 || appArrOffset == 0 {
		return false
	}
	if appArrSize > maxSharedMemoryAppEntries {
		return false
	}

	region := uint64(regionSize)
	offset := uint64(appArrOffset)
	entrySize := uint64(appEntrySize)
	entryCount := uint64(appArrSize)
	if offset >= region || entrySize == 0 {
		return false
	}

	available := region - offset
	return entryCount <= available/entrySize
}
