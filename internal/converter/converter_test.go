package converter

import "testing"

func TestValidatePair(t *testing.T) {
	tests := []struct {
		name      string
		inputPath string
		target    string
		wantErr   bool
	}{
		{name: "pdf to epub", inputPath: "book.pdf", target: "epub"},
		{name: "epub to pdf", inputPath: "book.epub", target: "pdf"},
		{name: "invalid", inputPath: "book.pdf", target: "pdf", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePair(tc.inputPath, tc.target)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
