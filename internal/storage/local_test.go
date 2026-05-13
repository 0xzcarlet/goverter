package storage

import "testing"

func TestAbsPath(t *testing.T) {
	local := New("/srv/file-converter/data", "/srv/file-converter/data/uploads", "/srv/file-converter/data/outputs", "/srv/file-converter/data/tmp")
	got := local.AbsPath("uploads/demo.pdf")
	want := "/srv/file-converter/data/uploads/demo.pdf"
	if got != want {
		t.Fatalf("AbsPath() = %q, want %q", got, want)
	}
}
