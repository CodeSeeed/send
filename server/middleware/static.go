package middleware

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFiles serves the built frontend (web/dist) for non-API requests.
// It is registered as the NoRoute handler, so it only runs when no /api
// route matches. Behavior:
//   - Existing files (assets, favicon, …) are served directly. Vite-hashed
//     assets under /assets/ get a long immutable cache; index.html etc. get
//     no-cache so new builds are picked up promptly.
//   - Any other path falls back to index.html (SPA). The app uses hash
//     routing, so the browser always loads "/" and client-side navigation
//     never changes the path — the fallback is for robustness.
//   - Unknown /api/* paths return JSON 404 instead of the HTML page.
//
// Path traversal is rejected explicitly and the resolved path is verified to
// stay inside staticDir before anything is served.
func StaticFiles(staticDir string) gin.HandlerFunc {
	absStatic, err := filepath.Abs(staticDir)
	if err != nil {
		log.Fatalf("static_dir 解析失败: %v", err)
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Never render HTML for unknown API routes
		if strings.HasPrefix(path, "/api/") || path == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "接口不存在"})
			return
		}

		// Reject path traversal attempts up front
		if strings.Contains(path, "..") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Resolve the requested file and verify it stays inside staticDir
		filePath := filepath.Join(absStatic, filepath.FromSlash(strings.TrimPrefix(path, "/")))
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if absFile != absStatic && !strings.HasPrefix(absFile, absStatic+string(os.PathSeparator)) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Serve the file directly if it exists (and is not a directory listing)
		if info, err := os.Stat(absFile); err == nil && !info.IsDir() {
			setCacheHeaders(c, path)
			c.File(absFile)
			return
		}

		// SPA fallback: index.html
		indexPath := filepath.Join(absStatic, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.File(indexPath)
	}
}

func setCacheHeaders(c *gin.Context, path string) {
	if strings.HasPrefix(path, "/assets/") {
		// Vite emits content-hashed filenames under /assets/: cache forever
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
}