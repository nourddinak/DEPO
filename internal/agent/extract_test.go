package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractArchiveSafe(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "safe.tar.gz")
	destDir := filepath.Join(tempDir, "extracted")

	// Create a safe archive
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := "Hello DEPO world"
	_ = tw.WriteHeader(&tar.Header{
		Name: "sub/index.html",
		Mode: 0644,
		Size: int64(len(content)),
	})
	_, _ = tw.Write([]byte(content))

	_ = tw.Close()
	_ = gzw.Close()
	_ = os.WriteFile(archivePath, buf.Bytes(), 0644)

	// Extract
	if err := ExtractArchive(archivePath, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	extractedFile := filepath.Join(destDir, "sub", "index.html")
	data, err := os.ReadFile(extractedFile)
	if err != nil || string(data) != content {
		t.Fatalf("Extracted file mismatch: %v, got %s", err, string(data))
	}
}

func TestExtractArchiveZipSlipRejection(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "malicious.tar.gz")
	destDir := filepath.Join(tempDir, "target")

	// Create malicious archive attempting to escape root via ../../escape.txt
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := "Malicious Payload"
	_ = tw.WriteHeader(&tar.Header{
		Name: "../../escape.txt",
		Mode: 0644,
		Size: int64(len(content)),
	})
	_, _ = tw.Write([]byte(content))

	_ = tw.Close()
	_ = gzw.Close()
	_ = os.WriteFile(archivePath, buf.Bytes(), 0644)

	// Attempt extraction
	err := ExtractArchive(archivePath, destDir)
	if err == nil {
		t.Fatal("Expected ExtractArchive to reject malicious Zip-Slip path, but it succeeded!")
	}
	if !errors.Is(err, ErrZipSlip) && !strings.Contains(err.Error(), "traversing outside") {
		t.Fatalf("Expected ErrZipSlip error, got: %v", err)
	}
}
