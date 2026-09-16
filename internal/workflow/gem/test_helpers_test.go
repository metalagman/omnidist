package gem

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func gzipWriter(t *testing.T, w io.Writer) *gzip.Writer {
	t.Helper()
	return gzip.NewWriter(w)
}

func tarWriter(t *testing.T, w io.Writer) *tar.Writer {
	t.Helper()
	return tar.NewWriter(w)
}

func writeTarEntry(t *testing.T, tw *tar.Writer, name string, data []byte) {
	t.Helper()
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}); err != nil {
		t.Fatalf("WriteHeader(%q) error = %v", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("Write(%q) error = %v", name, err)
	}
}
