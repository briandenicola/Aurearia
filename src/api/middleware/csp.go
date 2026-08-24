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

// backgroundRemovalWorkerPathPrefix is the exact static path namespace Vite
// emits the background-removal module worker under (see
// src/web/vite.config.ts worker.rollupOptions.output.entryFileNames /
// chunkFileNames: "assets/workers/[name]-[hash].js"). The trailing slash is
// load-bearing: it is what keeps "/assets/workers-evil/..." (a different,
// hypothetical asset directory that merely shares a prefix) from matching —
// strings.HasPrefix compares byte-for-byte, and "assets/workers-evil"[15] is
// '-', not '/'.
const backgroundRemovalWorkerPathPrefix = "/assets/workers/"

// workerScriptCSP is the policy for exactly one response path namespace:
// the background-removal module worker's own script/chunk files. It is the
// only policy in this file that carries 'unsafe-eval', and it is scoped
// there deliberately.
//
// Why 'unsafe-eval' is needed at all: @imgly/background-removal bundles
// ndarray, whose compileConstructor() calls `new Function(...)` to build a
// typed-array constructor at runtime. Under a strict script-src this throws
// `EvalError: Evaluating a string as JavaScript violates ... script-src`.
// Aurelia proved the cause and the fix experimentally (commit fbbe8503,
// .squad/agents/aurelia/history.md): a strict CSP on the worker's own
// response reproduces the exact production EvalError; granting
// 'unsafe-eval' only to that response makes @imgly/background-removal
// complete successfully, with the page's own CSP (appCSP) left untouched.
//
// Why scoping this to a worker response is a real isolation boundary and
// not just a narrower place to put the same risk: a dedicated same-origin
// module Worker has no `window`, no `document`, no DOM, and does not share
// the page's storage areas — `localStorage`/`sessionStorage` are not
// reachable from WorkerGlobalScope, and the worker never receives the SPA's
// JWT (see backgroundRemovalWorker.ts: the only thing posted into it is an
// image Blob, and the only thing posted out is a result Blob or a plain
// {name, message} error). So even though 'unsafe-eval' would normally let
// injected script exfiltrate the access token via eval'd code, there is no
// token, no cookie jar, and no document to inject into inside this
// execution context for that eval to act on. Directives that only matter to
// a document — style-src, img-src, font-src, base-uri, form-action,
// frame-ancestors, manifest-src — are omitted entirely rather than
// inherited from appCSP, so default-src 'none' is the true fallback for
// everything not explicitly listed below (least privilege: nothing is
// implicitly allowed just because it was allowed on the page).
//
//   - script-src 'self' covers the worker's own static ES-module imports
//     (backgroundRemovalWorker.ts plus the bundled @imgly/background-removal
//     and onnxruntime-web chunks, all emitted alongside it under
//     assets/workers/ per vite.config.ts chunkFileNames); 'unsafe-eval' is
//     ndarray's compileConstructor(); 'wasm-unsafe-eval' is required to
//     compile the ONNX model to WebAssembly, mirroring appCSP; blob: is kept
//     for parity with appCSP's script-src in case the vendored glue code
//     dynamically imports a blob:-backed module here too.
//   - connect-src 'self' allows fetching the model/quantized-weight chunks
//     served same-origin under /imgly-background-removal/ (see
//     backgroundRemovalConfig.ts's publicPath); blob: covers
//     onnxruntime-web handing back fetched model bytes as a blob: response.
//   - worker-src 'self' blob: covers onnxruntime-web spawning its own nested
//     thread worker from a blob: URL for WASM threading, the same allowance
//     appCSP already carries for the outer worker.
//   - object-src 'none' — no plugin content belongs in a worker, made
//     explicit even though default-src 'none' already implies it.
var workerScriptCSP = strings.Join([]string{
	"default-src 'none'",
	"script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval' blob:",
	"connect-src 'self' blob:",
	"worker-src 'self' blob:",
	"object-src 'none'",
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
		switch {
		case strings.HasPrefix(c.Request.URL.Path, "/swagger"):
			c.Header("Content-Security-Policy", swaggerCSP)
		case strings.HasPrefix(c.Request.URL.Path, backgroundRemovalWorkerPathPrefix):
			c.Header("Content-Security-Policy", workerScriptCSP)
		default:
			c.Header("Content-Security-Policy", appCSP)
		}
		c.Next()
	}
}
