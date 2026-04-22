package main

import "sync"

var (
	sharedWebAPIMu   sync.Mutex
	sharedWebAPI     *WebAPI
	sharedWebAPIRefs int
)

func AcquireSharedWebAPI(cfg *MonitorConfig) (*WebAPI, error) {
	sharedWebAPIMu.Lock()
	defer sharedWebAPIMu.Unlock()

	if sharedWebAPI == nil {
		runtime, err := NewWebAPI(cfg)
		if err != nil {
			return nil, err
		}
		sharedWebAPI = runtime
		sharedWebAPIRefs = 1
		return sharedWebAPI, nil
	}

	if cfg != nil {
		if err := sharedWebAPI.ApplyConfig(cfg); err != nil {
			return nil, err
		}
	}
	sharedWebAPIRefs++
	return sharedWebAPI, nil
}

func ReleaseSharedWebAPI(runtime *WebAPI) {
	if runtime == nil {
		return
	}

	var toClose *WebAPI
	sharedWebAPIMu.Lock()
	if sharedWebAPI == runtime {
		if sharedWebAPIRefs > 0 {
			sharedWebAPIRefs--
		}
		if sharedWebAPIRefs == 0 {
			toClose = sharedWebAPI
			sharedWebAPI = nil
		}
	}
	sharedWebAPIMu.Unlock()

	if toClose != nil {
		toClose.Close()
	}
}

func ApplyConfigToSharedWebAPI(cfg *MonitorConfig) error {
	if cfg == nil {
		return nil
	}

	sharedWebAPIMu.Lock()
	runtime := sharedWebAPI
	sharedWebAPIMu.Unlock()
	if runtime == nil {
		return nil
	}
	return runtime.ApplyConfig(cfg)
}
