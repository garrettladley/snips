package snips_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/garrettladley/snips"
)

func TestPackageNameSameAsDirectory(t *testing.T) {
	dir := createTempDir(t)
	filePath := filepath.Join(dir, "views", "foo", "bar.templ")
	createTempFile(t, filePath, "package foo\n\ntempl Hello(name string) {\n  <div>Hello, { name }</div>\n}")

	pkg := snips.PackageName(filepath.Join(dir, "views", "foo"))
	if pkg != "foo" {
		t.Fatalf("expected package name to be 'foo', got '%s'", pkg)
	}
}

func TestPackageNameDifferentFromDirectory(t *testing.T) {
	dir := createTempDir(t)
	filePath := filepath.Join(dir, "views", "foo", "bar.templ")
	createTempFile(t, filePath, "package bar\n\ntempl Hello(name string) {\n  <div>Hello, { name }</div>\n}")

	pkg := snips.PackageName(filepath.Join(dir, "views", "foo"))
	if pkg != "bar" {
		t.Fatalf("expected package name to be 'bar', got '%s'", pkg)
	}
}

func TestPackageNameFallback(t *testing.T) {
	dir := createTempDir(t)
	filePath := filepath.Join(dir, "views", "foo", "ex.rs")
	createTempFile(t, filePath, "fn main() {\n    println!(\"Hello World!\");\n}")

	pkg := snips.PackageName(filepath.Join(dir, "views", "foo"))
	if pkg != "foo" {
		t.Fatalf("expected package name to be 'foo', got '%s'", pkg)
	}
}

func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "snips")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func createTempFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
