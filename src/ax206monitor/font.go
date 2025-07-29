package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type FontCache struct {
	titleFont   font.Face
	contentFont font.Face
	smallFont   font.Face
	largeFont   font.Face
	headerFont  font.Face
	fontMap     map[int]font.Face
	mutex       sync.RWMutex
}

var (
	globalFontCache *FontCache
	fontCacheMutex  sync.RWMutex
	fontCacheLoaded bool
)

func loadFontCache() (*FontCache, error) {
	fontCacheMutex.RLock()
	if fontCacheLoaded && globalFontCache != nil {
		cache := globalFontCache
		fontCacheMutex.RUnlock()
		return cache, nil
	}
	fontCacheMutex.RUnlock()

	fontCacheMutex.Lock()
	defer fontCacheMutex.Unlock()

	if fontCacheLoaded && globalFontCache != nil {
		return globalFontCache, nil
	}

	cache := &FontCache{
		fontMap: make(map[int]font.Face),
	}

	loadedFont := findSystemFont()
	if loadedFont == "" {
		logWarnModule("font", "No suitable font found, using system default")
	} else {
		logInfoModule("font", "Using font: %s", filepath.Base(loadedFont))
	}

	var err error
	cache.titleFont, err = gg.LoadFontFace(loadedFont, 18)
	if err != nil {
		cache.titleFont, _ = gg.LoadFontFace("", 18)
	}

	cache.contentFont, err = gg.LoadFontFace(loadedFont, 16)
	if err != nil {
		cache.contentFont, _ = gg.LoadFontFace("", 16)
	}

	cache.smallFont, err = gg.LoadFontFace(loadedFont, 16)
	if err != nil {
		cache.smallFont, _ = gg.LoadFontFace("", 16)
	}

	cache.largeFont, err = gg.LoadFontFace(loadedFont, 18)
	if err != nil {
		cache.largeFont, _ = gg.LoadFontFace("", 18)
	}

	cache.headerFont, err = gg.LoadFontFace(loadedFont, 16)
	if err != nil {
		cache.headerFont, _ = gg.LoadFontFace("", 16)
	}

	globalFontCache = cache
	fontCacheLoaded = true

	return cache, nil
}

func findSystemFont() string {

	fontFiles := []string{
		"wqy-microhei.ttc",
		"wqy-zenhei.ttc",
		"NotoSansCJK-Regular.ttc",
		"SourceHanSansSC-Regular.otf",
		"msyh.ttc",
		"simhei.ttf",
		"Roboto-Regular.ttf",
		"Ubuntu-Regular.ttf",
		"DejaVuSans.ttf",
		"LiberationSans-Regular.ttf",
		"arial.ttf",
		"Arial.ttf",
		"helvetica.ttf",
		"FreeSans.ttf",
	}

	for _, fontFile := range fontFiles {
		if font := findFontByName([]string{fontFile}); font != "" {
			return font
		}
	}

	return ""
}

func findFontByName(fontNames []string) string {
	fontDirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		"/System/Library/Fonts",
		"/Library/Fonts",
		"~/.fonts",
		"~/.local/share/fonts",
	}

	for _, fontName := range fontNames {
		for _, dir := range fontDirs {
			cmd := exec.Command("find", dir, "-name", "*"+fontName+"*", "-type", "f")
			output, err := cmd.Output()
			if err != nil {
				continue
			}

			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				if line != "" && (strings.HasSuffix(line, ".ttf") ||
					strings.HasSuffix(line, ".ttc") ||
					strings.HasSuffix(line, ".otf")) {
					if _, err := gg.LoadFontFace(line, 16); err == nil {
						return line
					}
				}
			}
		}
	}
	return ""
}

func (fc *FontCache) GetFont(size int) (font.Face, error) {
	fc.mutex.RLock()
	if fc.fontMap == nil {
		fc.mutex.RUnlock()
		fc.mutex.Lock()
		if fc.fontMap == nil {
			fc.fontMap = make(map[int]font.Face)
		}
		fc.mutex.Unlock()
		fc.mutex.RLock()
	}

	if face, exists := fc.fontMap[size]; exists {
		fc.mutex.RUnlock()
		return face, nil
	}
	fc.mutex.RUnlock()

	face, err := gg.LoadFontFace(findSystemFont(), float64(size))
	if err != nil {
		return fc.contentFont, err
	}

	fc.mutex.Lock()
	fc.fontMap[size] = face
	fc.mutex.Unlock()

	return face, nil
}
