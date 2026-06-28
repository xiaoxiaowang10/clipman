package clipman_test

import (
	"path/filepath"
	"testing"

	. "clipman"
)

func TestExeDir(t *testing.T) {
	dir := ExeDir()
	if dir == "" {
		t.Fatal("ExeDir returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("ExeDir returned relative path: %q", dir)
	}
}
