package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

type staticHost struct {
	mode       string
	wwwDist    string
	adminDist  string
	wwwProxy   *httputil.ReverseProxy
	adminProxy *httputil.ReverseProxy
}

func newStaticHost() (*staticHost, error) {
	host := &staticHost{
		mode:      "release",
		wwwDist:   "./assets/www",
		adminDist: "./assets/admin",
	}

	wwwDev := os.Getenv("OPENFI_WWW_DEV_SERVER")
	adminDev := os.Getenv("OPENFI_ADMIN_DEV_SERVER")
	if wwwDev != "" && adminDev != "" {
		host.mode = "dev"
		wwwProxy, err := buildReverseProxy(wwwDev, "")
		if err != nil {
			return nil, err
		}
		adminProxy, err := buildReverseProxy(adminDev, "")
		if err != nil {
			return nil, err
		}
		host.wwwProxy = wwwProxy
		host.adminProxy = adminProxy
	}

	return host, nil
}

func buildReverseProxy(targetURL string, stripPrefix string) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, errors.New("invalid dev server url")
	}

	base := httputil.NewSingleHostReverseProxy(u)
	originalDirector := base.Director
	base.Director = func(req *http.Request) {
		originalDirector(req)
		if stripPrefix != "" && strings.HasPrefix(req.URL.Path, stripPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
		req.Host = u.Host
	}
	return base, nil
}

func (h *staticHost) registerRoutes(e *echo.Echo) {
	e.Any("/admin", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/admin/")
	})
	e.Any("/admin/*", h.handleAdminStatic)
	e.GET("/", h.handleWWWStatic)
	e.Any("/*", h.handleWWWStatic)
}

func (h *staticHost) handleAdminStatic(c echo.Context) error {
	path := c.Request().URL.Path
	if path == "/admin/api" || strings.HasPrefix(path, "/admin/api/") {
		return echo.ErrNotFound
	}
	if h.mode == "dev" && h.adminProxy != nil {
		h.adminProxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
	return serveReleaseSPA(c, h.adminDist, strings.TrimPrefix(c.Request().URL.Path, "/admin"))
}

func (h *staticHost) handleWWWStatic(c echo.Context) error {
	path := c.Request().URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") || path == "/admin/api" || strings.HasPrefix(path, "/admin/api/") {
		return echo.ErrNotFound
	}
	if strings.HasPrefix(path, "/admin/") {
		return echo.ErrNotFound
	}

	if h.mode == "dev" && h.wwwProxy != nil {
		h.wwwProxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
	return serveReleaseSPA(c, h.wwwDist, path)
}

func serveReleaseSPA(c echo.Context, root string, reqPath string) error {
	normalized := strings.TrimPrefix(filepath.Clean("/"+reqPath), "/")
	if normalized == "." || normalized == "" {
		return c.File(filepath.Join(root, "index.html"))
	}

	target := filepath.Clean(filepath.Join(root, normalized))
	rootClean := filepath.Clean(root)
	if target != rootClean && !strings.HasPrefix(target, rootClean+string(os.PathSeparator)) {
		return echo.ErrForbidden
	}

	stat, err := os.Stat(target)
	if err == nil && !stat.IsDir() {
		return c.File(target)
	}
	if err == nil && stat.IsDir() {
		indexPath := filepath.Join(target, "index.html")
		if _, indexErr := os.Stat(indexPath); indexErr == nil {
			return c.File(indexPath)
		}
	}
	return c.File(filepath.Join(root, "index.html"))
}
