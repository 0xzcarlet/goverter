package storage

import (
	"errors"
	"testing"
)

func TestAbsPath(t *testing.T) {
	local := New("/srv/file-converter/data", "/srv/file-converter/data/uploads", "/srv/file-converter/data/outputs", "/srv/file-converter/data/tmp")
	got := local.AbsPath("uploads/demo.pdf")
	want := "/srv/file-converter/data/uploads/demo.pdf"
	if got != want {
		t.Fatalf("AbsPath() = %q, want %q", got, want)
	}
}

func TestResolvePathRejectsTraversal(t *testing.T) {
	local := New("/srv/file-converter/data", "/srv/file-converter/data/uploads", "/srv/file-converter/data/outputs", "/srv/file-converter/data/tmp")
	_, err := local.resolvePath("../secrets.txt")
	if !errors.Is(err, ErrInvalidStorageKey) {
		t.Fatalf("resolvePath() error = %v, want %v", err, ErrInvalidStorageKey)
	}
}
