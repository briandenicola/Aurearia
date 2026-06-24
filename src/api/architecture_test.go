package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var allowedServiceGORMFiles = map[string]string{
	"auction_lot_service.go":         "transaction orchestration only; queries stay in repositories via WithTx",
	"coin_intake_service.go":         "transaction orchestration only; queries stay in repositories via WithTx",
	"coin_service.go":                "transaction orchestration only; queries stay in repositories via WithTx",
	"collection_tools_service.go":    "legacy proposal commit transaction still performs field/tag writes; tracked boundary debt",
	"reference_migration_service.go": "legacy migration reads/writes denormalized data directly; tracked boundary debt",
}

func TestArchitecture(t *testing.T) {
	t.Run("no direct database imports", TestNoDirectDatabaseImports)
	t.Run("handlers do not use raw SQL", TestHandlersDoNotUseRawSQL)
	t.Run("handlers do not use GORM", TestHandlersDoNotUseGORM)
	t.Run("handlers do not mention GORM types", TestHandlersDoNotUseGORMTextPatterns)
	t.Run("package import matrix", TestPackageImportMatrix)
}

// TestNoDirectDatabaseImports ensures that only main.go imports the database
// package. All other packages must receive their *gorm.DB via dependency
// injection (constructor parameters). This prevents regression to the old
// pattern of scattering database.DB calls throughout handlers and services.
func TestNoDirectDatabaseImports(t *testing.T) {
	forbiddenImport := "github.com/briandenicola/ancient-coins-api/database"

	// Directories that must NOT import the database package directly
	restricted := []string{
		"handlers",
		"services",
		"middleware",
		"repository",
	}

	for _, dir := range restricted {
		dirPath := filepath.Join(".", dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			t.Fatalf("Failed to read directory %s: %v", dir, err)
		}

		fset := token.NewFileSet()
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}

			filePath := filepath.Join(dirPath, entry.Name())
			f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", filePath, err)
			}

			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if importPath == forbiddenImport {
					t.Errorf(
						"%s imports %q directly. Use dependency injection instead — "+
							"accept *gorm.DB or a repository as a constructor parameter. "+
							"Only main.go should reference the database package.",
						filePath, forbiddenImport,
					)
				}
			}
		}
	}
}

// TestHandlersDoNotUseRawSQL checks that handler files do not contain raw SQL
// query strings, which should live in the repository layer.
func TestHandlersDoNotUseRawSQL(t *testing.T) {
	handlersDir := filepath.Join(".", "handlers")
	if _, err := os.Stat(handlersDir); os.IsNotExist(err) {
		t.Skip("handlers directory not found")
	}

	sqlPatterns := []string{
		"SELECT ",
		"INSERT INTO",
		"UPDATE ",
		"DELETE FROM",
		".Raw(",
		".Exec(",
		".Model(&",
	}

	// These are false-positive patterns that appear in non-SQL contexts
	allowList := []string{
		"swagger_types.go",
	}

	entries, err := os.ReadDir(handlersDir)
	if err != nil {
		t.Fatalf("Failed to read handlers directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		allowed := false
		for _, a := range allowList {
			if entry.Name() == a {
				allowed = true
				break
			}
		}
		if allowed {
			continue
		}

		filePath := filepath.Join(handlersDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", filePath, err)
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip comments and string constants (prompts, etc.)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}

			for _, pattern := range sqlPatterns {
				if strings.Contains(line, pattern) {
					// Ignore if inside a backtick or quote string constant (prompts contain SQL-like words)
					if isInsideStringConstant(lines, lineNum) {
						continue
					}
					t.Errorf(
						"%s:%d contains raw SQL pattern %q. "+
							"SQL queries belong in the repository layer, not handlers.",
						filePath, lineNum+1, pattern,
					)
				}
			}
		}
	}
}

// TestHandlersDoNotUseGORM ensures handlers never own database/GORM access.
func TestHandlersDoNotUseGORM(t *testing.T) {
	handlersDir := filepath.Join(".", "handlers")
	if _, err := os.Stat(handlersDir); os.IsNotExist(err) {
		t.Skip("handlers directory not found")
	}

	forbiddenPatterns := []string{
		".DB(",
		".Where(",
		".Find(",
		".First(",
		".Scan(",
		".Count(",
		".Model(",
		".Raw(",
		".Exec(",
	}

	entries, err := os.ReadDir(handlersDir)
	if err != nil {
		t.Fatalf("Failed to read handlers directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filePath := filepath.Join(handlersDir, entry.Name())
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			t.Fatalf("Failed to parse %s: %v", filePath, err)
		}

		importedNames := importedIdentifierNames(f)
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == "gorm.io/gorm" {
				t.Errorf(
					"%s imports GORM directly. Handlers must delegate database access to repositories/services.",
					filePath,
				)
			}
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pattern := "." + sel.Sel.Name + "("
			if !containsString(forbiddenPatterns, pattern) {
				return true
			}
			if root := rootIdentifier(sel.X); root != "" && importedNames[root] {
				return true
			}

			pos := fset.Position(sel.Pos())
			t.Errorf(
				"%s:%d contains GORM/direct DB pattern %q. Handlers must delegate database access to repositories/services.",
				filePath, pos.Line, pattern,
			)
			return true
		})
	}
}

func importedIdentifierNames(f *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range f.Imports {
		if imp.Name != nil {
			if imp.Name.Name != "_" && imp.Name.Name != "." {
				names[imp.Name.Name] = true
			}
			continue
		}
		importPath := strings.Trim(imp.Path.Value, `"`)
		parts := strings.Split(importPath, "/")
		if len(parts) > 0 {
			names[parts[len(parts)-1]] = true
		}
	}
	return names
}

func rootIdentifier(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return rootIdentifier(e.X)
	case *ast.CallExpr:
		return rootIdentifier(e.Fun)
	case *ast.StarExpr:
		return rootIdentifier(e.X)
	case *ast.ParenExpr:
		return rootIdentifier(e.X)
	default:
		return ""
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestHandlersDoNotUseGORMTextPatterns(t *testing.T) {
	handlersDir := filepath.Join(".", "handlers")
	if _, err := os.Stat(handlersDir); os.IsNotExist(err) {
		t.Skip("handlers directory not found")
	}

	forbiddenPatterns := []string{
		"*gorm.DB",
		"gorm.",
	}

	entries, err := os.ReadDir(handlersDir)
	if err != nil {
		t.Fatalf("Failed to read handlers directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filePath := filepath.Join(handlersDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", filePath, err)
		}
		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || isInsideStringConstant(lines, lineNum) {
				continue
			}
			for _, pattern := range forbiddenPatterns {
				if strings.Contains(line, pattern) {
					t.Errorf(
						"%s:%d contains GORM/direct DB pattern %q. Handlers must delegate database access to repositories/services.",
						filePath, lineNum+1, pattern,
					)
				}
			}
		}
	}
}

// isInsideStringConstant is a heuristic to check if a line is inside a
// multi-line backtick string (used for prompts). It counts backticks
// before the line — an odd count means we're inside a raw string.
func isInsideStringConstant(lines []string, targetLine int) bool {
	backtickCount := 0
	for i := 0; i < targetLine; i++ {
		backtickCount += strings.Count(lines[i], "`")
	}
	return backtickCount%2 == 1
}

// TestPackageImportMatrix enforces the layered architecture import rules:
//   - handlers/ → services/, repository/, models/ (+ gin, standard lib)
//   - services/ → repository/, models/ (+ standard lib, NO gin, NO handlers)
//   - repository/ → models/ (+ gorm, standard lib)
//   - models/ → standard library only
func TestPackageImportMatrix(t *testing.T) {
	modulePrefix := "github.com/briandenicola/ancient-coins-api/"

	// allowedInternal defines which internal packages each layer may import.
	allowedInternal := map[string][]string{
		"handlers":   {"services", "repository", "models", "capture"},
		"services":   {"repository", "models"},
		"repository": {"models"},
		"models":     {}, // no internal imports
	}

	// allowedExternalPrefixes defines non-stdlib prefixes each layer may use.
	allowedExternalPrefixes := map[string][]string{
		"handlers":   {"github.com/gin-gonic/gin", "github.com/go-webauthn/webauthn", "github.com/go-pdf/fpdf", "golang.org/x/crypto", "golang.org/x/net", "gopkg.in/yaml.v3"},
		"services":   {"github.com/coreos/go-oidc/v3", "github.com/golang-jwt/jwt", "golang.org/x/crypto", "golang.org/x/net", "golang.org/x/oauth2", "golang.org/x/text"},
		"repository": {"gorm.io/gorm"},
		"models":     {},
	}

	for dir, allowedPkgs := range allowedInternal {
		dirPath := filepath.Join(".", dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			t.Fatalf("Failed to read directory %s: %v", dir, err)
		}

		fset := token.NewFileSet()
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			// Skip test files — they legitimately import test helpers
			if strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}

			filePath := filepath.Join(dirPath, entry.Name())
			f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", filePath, err)
			}

			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)

				// Standard library imports are always OK
				if isStdLib(importPath) {
					continue
				}

				// Check internal project imports
				if strings.HasPrefix(importPath, modulePrefix) {
					pkg := strings.TrimPrefix(importPath, modulePrefix)
					// Strip any sub-path (e.g. "models/foo" → "models")
					if idx := strings.Index(pkg, "/"); idx != -1 {
						pkg = pkg[:idx]
					}
					if !contains(allowedPkgs, pkg) {
						t.Errorf(
							"%s imports internal package %q which violates the layer rules. "+
								"%s/ may only import: %v",
							filePath, importPath, dir, allowedPkgs,
						)
					}
					continue
				}

				if dir == "services" && importPath == "gorm.io/gorm" {
					if rationale := allowedServiceGORMFiles[entry.Name()]; rationale != "" {
						continue
					}
					t.Errorf(
						"%s imports gorm.io/gorm without a documented service-layer exception. "+
							"Move GORM access to repository/ or document the temporary exception in allowedServiceGORMFiles.",
						filePath,
					)
					continue
				}

				// Check external (third-party) imports
				allowed := false
				for _, prefix := range allowedExternalPrefixes[dir] {
					if strings.HasPrefix(importPath, prefix) {
						allowed = true
						break
					}
				}
				if !allowed {
					t.Errorf(
						"%s imports external package %q which is not in the allowed list for %s/. "+
							"Allowed external prefixes: %v",
						filePath, importPath, dir, allowedExternalPrefixes[dir],
					)
				}
			}
		}
	}
}

func isStdLib(importPath string) bool {
	// Standard library packages don't contain a dot in the first path segment
	if i := strings.Index(importPath, "/"); i != -1 {
		return !strings.Contains(importPath[:i], ".")
	}
	return !strings.Contains(importPath, ".")
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
