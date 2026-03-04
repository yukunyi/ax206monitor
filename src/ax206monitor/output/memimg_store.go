package output

import "sync"

var memImgStore = struct {
	mutex sync.RWMutex
	png   []byte
}{}

func SetMemImgPNG(data []byte) {
	memImgStore.mutex.Lock()
	defer memImgStore.mutex.Unlock()
	if len(data) == 0 {
		memImgStore.png = nil
		return
	}
	memImgStore.png = append([]byte(nil), data...)
}

func GetMemImgPNG() ([]byte, bool) {
	memImgStore.mutex.RLock()
	defer memImgStore.mutex.RUnlock()
	if len(memImgStore.png) == 0 {
		return nil, false
	}
	return append([]byte(nil), memImgStore.png...), true
}
