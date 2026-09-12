package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// failingReader yields n bytes of content, then fails every subsequent Read
// — simulating an io.Copy that dies partway through a real upload stream.
type failingReader struct{ remaining int }

func (f *failingReader) Read(p []byte) (int, error) {
	if f.remaining <= 0 {
		return 0, errors.New("simulated mid-stream read failure")
	}
	n := f.remaining
	if n > len(p) {
		n = len(p)
	}
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	f.remaining -= n
	return n, nil
}

// A failed copy must not leave the file OpenFile created on disk: nothing
// else ever revisits a generated name, so an uncleaned partial file is
// orphaned bytes with no code path that reads, retries, or removes it.
func TestCopyToFileRemovesPartialFileOnCopyFailure(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "partial.bin")

	if err := copyToFile(&failingReader{remaining: 10}, dest); err == nil {
		t.Fatal("expected copyToFile to return the underlying read error")
	}

	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("partial file must be removed after a copy failure; stat error: %v", statErr)
	}
}
