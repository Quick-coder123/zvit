package launch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShadowCopy(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "potik.exe")
	os.WriteFile(exe, []byte("build one"), 0o755)
	cache := filepath.Join(dir, "cache")

	a, err := ShadowCopy(exe, cache)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(a); string(b) != "build one" {
		t.Fatal("copy content")
	}
	// same build -> same copy
	if a2, _ := ShadowCopy(exe, cache); a2 != a {
		t.Fatal("same build must reuse the copy")
	}
	// drop a new version into the folder
	os.WriteFile(exe, []byte("build two"), 0o755)
	b, err := ShadowCopy(exe, cache)
	if err != nil || b == a {
		t.Fatalf("new build not copied: %v", err)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Fatal("old copy should be cleaned up")
	}
	if filepath.Ext(b) != ".exe" {
		t.Fatal("extension lost")
	}
}

func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "potik.exe")
	os.WriteFile(exe, []byte("v1"), 0o755)
	w := NewWatcher(exe)
	if w.Changed() {
		t.Fatal("unchanged file reported as changed")
	}
	os.WriteFile(exe, []byte("v2, longer"), 0o755)
	old := time.Now().Add(-time.Minute)
	os.Chtimes(exe, old, old)
	w.next = time.Time{}
	if !w.Changed() {
		t.Fatal("replacement not noticed")
	}
}
