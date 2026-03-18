package webui

import (
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func RegisterDevProxy(e *echo.Echo, viteURL string) error {
	target, err := url.Parse(viteURL)
	if err != nil {
		return fmt.Errorf("invalid vite url: %w", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	e.Any("/*", echo.WrapHandler(proxy))
	return nil
}

func RegisterEmbeddedFrontend(e *echo.Echo, staticFS fs.FS) {
	e.GET("/*", func(c echo.Context) error {
		path := strings.TrimPrefix(c.Param("*"), "/")
		if path == "" {
			path = "index.html"
		}
		if strings.HasPrefix(path, "api/") {
			return c.NoContent(http.StatusNotFound)
		}

		contentPath := path
		if _, err := fs.Stat(staticFS, contentPath); err != nil {
			// SPA fallback.
			contentPath = "index.html"
		}

		data, err := fs.ReadFile(staticFS, contentPath)
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}

		contentType := mime.TypeByExtension(filepath.Ext(contentPath))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		return c.Blob(http.StatusOK, contentType, data)
	})
}
