package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		"handlers":   {"github.com/gin-gonic/gin", "github.com/go-webauthn/webauthn", "github.com/go-pdf/fpdf", "golang.org/x/crypto", "golang.org/x/net", "gorm.io/gorm"},
		"services":   {"gorm.io/gorm", "github.com/golang-jwt/jwt", "golang.org/x/crypto", "golang.org/x/net", "golang.org/x/text"},
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
