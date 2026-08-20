package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type registeredRoute struct {
	Method string
	Path   string
	File   string
	Line   int
}

// routeSourceFiles are every file that registers HTTP routes. main.go creates
// the root groups; the register*Routes functions in routes_*.go fill them in.
// A new routes_*.go file MUST be added here or its routes escape this gate --
// TestRouteSourceFilesCoverAllRouteRegistrations enforces that.
var routeSourceFiles = []string{
	"main.go",
	"bootstrap.go",
	"routes_public.go",
	"routes_protected.go",
	"routes_admin.go",
	"routes_tools.go",
	"routes_internal.go",
}

type openAPIDocument struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

var intentionallyUndocumentedRoutes = map[string]string{
	"GET /health":            "container orchestration health check, not part of the /api contract",
	"GET /healthz":           "container orchestration health check, not part of the /api contract",
	"GET /swagger/*any":      "Swagger UI asset route, not an API endpoint",
	"GET /uploads/*filepath": "root-level authenticated media alias; /api/uploads/{filepath} is the documented API route",
}

func TestRegisteredAPIRoutesAreDocumentedInOpenAPI(t *testing.T) {
	var routes []registeredRoute
	for _, file := range routeSourceFiles {
		parsed, err := parseRegisteredRoutes(file)
		if err != nil {
			t.Fatalf("parse registered routes from %s: %v", file, err)
		}
		routes = append(routes, parsed...)
	}
	// Guard against this gate silently going vacuous if the routes move again.
	if len(routes) < 100 {
		t.Fatalf("only %d routes parsed from %v; the route files have moved or the parser is broken", len(routes), routeSourceFiles)
	}
	operations, err := parseOpenAPIOperations("docs/swagger.json")
	if err != nil {
		t.Fatalf("parse OpenAPI operations: %v", err)
	}

	var missing []string
	for _, route := range routes {
		if isRouteIntentionallyUndocumented(route) {
			continue
		}
		openAPIPath := routeToOpenAPIPath(route.Path)
		key := route.Method + " " + openAPIPath
		if !operations[key] {
			missing = append(missing, fmt.Sprintf("%s %s (%s:%d, expected OpenAPI path %s)", route.Method, route.Path, route.File, route.Line, openAPIPath))
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("registered public API routes missing from OpenAPI:\n%s\n\nIf a route is intentionally non-public/internal, document it in intentionallyUndocumentedRoutes or isRouteIntentionallyUndocumented.", strings.Join(missing, "\n"))
	}
}

func parseRegisteredRoutes(filename string) ([]registeredRoute, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	// "r" (main.go) and "router" (bootstrap.go) are both the root *gin.Engine,
	// and "api" is the /api group. All three arrive as parameters or locals
	// whose prefix cannot be inferred from a Group(...) assignment in the file
	// being parsed, so they are seeded here.
	groupPrefixes := map[string]string{"r": "", "router": "", "api": "/api"}
	groupPattern := regexp.MustCompile(`^\s*(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"\)`)
	routePattern := regexp.MustCompile(`\b(\w+)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`)

	var routes []registeredRoute
	for lineNumber, line := range lines {
		if matches := groupPattern.FindStringSubmatch(line); matches != nil {
			group, parent, suffix := matches[1], matches[2], matches[3]
			groupPrefixes[group] = groupPrefixes[parent] + suffix
			continue
		}
		matches := routePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		receiver, method, path := matches[1], matches[2], matches[3]
		prefix, ok := groupPrefixes[receiver]
		if !ok {
			return nil, fmt.Errorf("%s line %d: route receiver %q has no known group prefix", filename, lineNumber+1, receiver)
		}
		routes = append(routes, registeredRoute{
			Method: method,
			Path:   prefix + path,
			File:   filename,
			Line:   lineNumber + 1,
		})
	}
	return routes, nil
}

func parseOpenAPIOperations(filename string) (map[string]bool, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var doc openAPIDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	operations := make(map[string]bool)
	for path, methods := range doc.Paths {
		for method := range methods {
			operations[strings.ToUpper(method)+" "+path] = true
		}
	}
	return operations, nil
}

func isRouteIntentionallyUndocumented(route registeredRoute) bool {
	if _, ok := intentionallyUndocumentedRoutes[route.Method+" "+route.Path]; ok {
		return true
	}
	return strings.HasPrefix(route.Path, "/api/internal/tools/")
}

func routeToOpenAPIPath(path string) string {
	path = strings.TrimPrefix(path, "/api")
	if path == "" {
		path = "/"
	}
	path = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`).ReplaceAllString(path, `{$1}`)
	path = regexp.MustCompile(`\*([A-Za-z_][A-Za-z0-9_]*)`).ReplaceAllString(path, `{$1}`)
	return path
}

// TestRouteSourceFilesCoverAllRouteRegistrations fails when a Go file registers
// HTTP routes but is not listed in routeSourceFiles. Without this, adding a
// routes_billing.go would silently exempt its endpoints from the OpenAPI drift
// gate above -- exactly what happened when the routes were first moved out of
// main.go, where the gate kept passing while seeing only one route.
func TestRouteSourceFilesCoverAllRouteRegistrations(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	listed := make(map[string]bool, len(routeSourceFiles))
	for _, f := range routeSourceFiles {
		listed[f] = true
	}

	routeCall := regexp.MustCompile(`\b\w+\.(GET|POST|PUT|DELETE|PATCH)\("`)

	var unlisted []string
	for _, entry := range entries {
		if strings.HasSuffix(entry, "_test.go") || listed[entry] {
			continue
		}
		data, err := os.ReadFile(entry)
		if err != nil {
			t.Fatalf("read %s: %v", entry, err)
		}
		if routeCall.Match(data) {
			unlisted = append(unlisted, entry)
		}
	}

	if len(unlisted) > 0 {
		sort.Strings(unlisted)
		t.Fatalf("these files register routes but are not in routeSourceFiles, so their routes skip the OpenAPI drift gate:\n  %s",
			strings.Join(unlisted, "\n  "))
	}
}

// TestRegisteredRouteCountIsPlausible pins the parsed route count so that a
// parser or layout change that quietly drops most routes fails loudly here
// rather than turning the drift gate into a no-op.
func TestRegisteredRouteCountIsPlausible(t *testing.T) {
	total := 0
	perFile := map[string]int{}
	for _, file := range routeSourceFiles {
		parsed, err := parseRegisteredRoutes(file)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		perFile[file] = len(parsed)
		total += len(parsed)
	}
	t.Logf("parsed routes: total=%d per-file=%v", total, perFile)

	if total < 100 {
		t.Fatalf("parsed only %d routes across %v; expected the full API surface", total, routeSourceFiles)
	}
	if perFile["routes_protected.go"] < 50 {
		t.Fatalf("routes_protected.go yielded only %d routes; it holds the bulk of the API", perFile["routes_protected.go"])
	}
}
