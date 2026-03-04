package output

import "strings"

const (
	TypeMemImg   = "memimg"
	TypeAX206USB = "ax206usb"
)

func NormalizeTypes(types []string) []string {
	normalized := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)

	for _, item := range types {
		typeName := strings.ToLower(strings.TrimSpace(item))
		switch typeName {
		case TypeMemImg, TypeAX206USB:
			if _, exists := seen[typeName]; exists {
				continue
			}
			seen[typeName] = struct{}{}
			normalized = append(normalized, typeName)
		}
	}

	if len(normalized) == 0 {
		return []string{TypeMemImg}
	}
	return normalized
}

func ResolveTypes(types []string, forceMemImg bool) []string {
	resolved := NormalizeTypes(types)
	if !forceMemImg {
		return resolved
	}
	for _, typeName := range resolved {
		if typeName == TypeMemImg {
			return resolved
		}
	}
	return append(resolved, TypeMemImg)
}

func BuildManager(types []string, forceMemImg bool) (*OutputManager, []string) {
	resolved := ResolveTypes(types, forceMemImg)
	manager := NewOutputManager()

	for _, typeName := range resolved {
		switch typeName {
		case TypeMemImg:
			manager.AddHandler(NewMemImgOutputHandler())
		case TypeAX206USB:
			handler, err := NewAX206USBOutputHandler()
			if err != nil {
				logErrorModule("ax206usb", "Handler creation failed: %v", err)
				continue
			}
			logInfoModule("ax206usb", "Handler ready")
			manager.AddHandler(handler)
		}
	}

	return manager, resolved
}
