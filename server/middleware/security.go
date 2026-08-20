package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers to every response.
// CSP restricts script/inline execution to mitigate XSS; frame-ancestors
// and X-Frame-Options prevent clickjacking; nosniff avoids MIME-sniffing.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer-when-downgrade")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				// blob: is required for in-browser previews: Share.vue fetches the
				// preview bytes with an Authorization header and renders them from
				// an object URL (PDF in an <iframe>, images in an <img>).
				"img-src 'self' data: blob:; "+
				"font-src 'self' data:; "+
				"frame-src 'self' blob:; "+
				"form-action 'self'; "+
				"base-uri 'none'; "+
				"frame-ancestors 'none'")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")
		c.Next()
	}
}
