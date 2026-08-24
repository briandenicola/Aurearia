package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// OSMTileHost is the OpenStreetMap tile origin used by the mint map
// (src/web/src/components/map/MintMapLeaflet.vue). It is the only external
// host the SPA contacts directly — every other remote image (auction lot
// photos, dealer listings) is fetched server-side through the image proxy and
// handed to the browser as a blob: URL, so it needs no allowance here.
const OSMTileHost = "https://*.tile.openstreetmap.org"

// appCSP is the policy for the SPA and the API.
//
// Notes on the non-obvious directives:
//
//   - 'wasm-unsafe-eval' is required by @imgly/background-removal, which
//     compiles an ONNX model to WebAssembly. It permits WASM compilation only;
//     it does NOT re-enable eval() or new Function() for JavaScript.
//   - script-src also allows blob: because that same library dynamically
//     imports its worker/WASM glue code as a module built from a blob: URL
//     (`import(URL.createObjectURL(...))`), which script-src governs
//     regardless of worker-src/connect-src already allowing blob:. This does
//     NOT permit inline or eval'd script — only script fetched from a blob
//     URL, which the app itself creates from its own bundled assets.
//   - style-src allows 'unsafe-inline' because Leaflet and the sanitized
//     Markdown pipeline both emit style attributes. DOMPurify strips script
//     vectors from that HTML, and CSS-only exfiltration is a much smaller risk
//     than shipping no policy at all. Scripts remain strictly 'self'.
//   - worker-src includes blob: for the background-removal worker.
//   - frame-ancestors 'none' duplicates the X-Frame-Options header for
//     browsers that honour CSP3 in preference to it.
var appCSP = strings.Join([]string{
	"default-src 'self'",
	"base-uri 'self'",
	"object-src 'none'",
	"frame-ancestors 'none'",
	"form-action 'self'",
	"script-src 'self' 'wasm-unsafe-eval' blob:",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data: blob: " + OSMTileHost,
	"font-src 'self' data:",
	"connect-src 'self' blob:",
	"worker-src 'self' blob:",
	"manifest-src 'self'",
}, "; ")

// swaggerCSP relaxes script/style for the bundled Swagger UI only, which
// inlines its own bootstrap script and stylesheet. Keeping it on a separate
// policy means the SPA never has to carry 'unsafe-inline' for scripts.
var swaggerCSP = strings.Join([]string{
	"default-src 'self'",
	"base-uri 'self'",
	"object-src 'none'",
	"frame-ancestors 'none'",
	"script-src 'self' 'unsafe-inline'",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data: blob:",
	"font-src 'self' data:",
	"connect-src 'self'",
}, "; ")

// ContentSecurityPolicy sets a Content-Security-Policy header on every
// response. The SPA stores its access token in web storage, so CSP is the
// control that keeps an injected script from reaching it.
func ContentSecurityPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			c.Header("Content-Security-Policy", swaggerCSP)
		} else {
			c.Header("Content-Security-Policy", appCSP)
		}
		c.Next()
	}
}
