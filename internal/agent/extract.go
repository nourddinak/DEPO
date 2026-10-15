package agent

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrZipSlip         = errors.New("archive contains illegal path traversing outside destination")
	ErrForbiddenSymlink = errors.New("archive contains illegal symlink escaping destination root")
	ErrArchiveTooLarge = errors.New("extracted archive size exceeded maximum safety threshold")
)

const maxExtractBytes = 500 * 1024 * 1024 // 500 MB limit

// ExtractArchive extracts either a .tar.gz or .zip file into destDir safely.
func ExtractArchive(archivePath, destDir string) error {
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination root: %w", err)
	}

	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return extractTarGz(archivePath, destDir)
	} else if strings.HasSuffix(lower, ".zip") {
		return extractZip(archivePath, destDir)
	}

	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var totalBytes int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Clean path and verify Zip-Slip protection
		cleanedName := filepath.Clean(header.Name)
		targetPath := filepath.Join(destDir, cleanedName)

		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(filepath.Separator)) && targetPath != filepath.Clean(destDir) {
			return fmt.Errorf("%w: %s", ErrZipSlip, header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode)&0777)
			if err != nil {
				return err
			}

			n, err := io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}

			totalBytes += n
			if totalBytes > maxExtractBytes {
				return ErrArchiveTooLarge
			}

		case tar.TypeSymlink:
			// Ensure symlink target does not escape destination directory
			linkTarget := header.Linkname
			if filepath.IsAbs(linkTarget) {
				return fmt.Errorf("%w: absolute symlink %s -> %s", ErrForbiddenSymlink, header.Name, linkTarget)
			}
			resolvedPath := filepath.Join(filepath.Dir(targetPath), linkTarget)
			if !strings.HasPrefix(filepath.Clean(resolvedPath), filepath.Clean(destDir)) {
				return fmt.Errorf("%w: symlink %s escapes destination", ErrForbiddenSymlink, header.Name)
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			_ = os.Remove(targetPath)
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	var totalBytes int64

	for _, file := range r.File {
		cleanedName := filepath.Clean(file.Name)
		targetPath := filepath.Join(destDir, cleanedName)

		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(filepath.Separator)) && targetPath != filepath.Clean(destDir) {
			return fmt.Errorf("%w: %s", ErrZipSlip, file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		n, err := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}

		totalBytes += n
		if totalBytes > maxExtractBytes {
			return ErrArchiveTooLarge
		}
	}

	return nil
}
