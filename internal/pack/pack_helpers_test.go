package pack_test

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
			t.Fatalf("create parent dir for %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("write file %q: %v", path, err)
		}
	}
}

func readArchiveEntries(t *testing.T, archivePath string) map[string]string {
	t.Helper()

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		return readTarGzArchiveEntries(t, archivePath)
	case strings.HasSuffix(archivePath, ".zip"):
		return readZIPArchiveEntries(t, archivePath)
	default:
		t.Fatalf("unsupported archive format for %q", archivePath)
		return nil
	}
}

func readZIPArchiveEntries(t *testing.T, archivePath string) map[string]string {
	t.Helper()

	r, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open archive %q: %v", archivePath, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Fatalf("close archive %q: %v", archivePath, err)
		}
	}()

	entries := make(map[string]string, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open archive entry %q: %v", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		closeErr := rc.Close()
		if err != nil {
			t.Fatalf("read archive entry %q: %v", f.Name, err)
		}
		if closeErr != nil {
			t.Fatalf("close archive entry %q: %v", f.Name, closeErr)
		}

		entries[f.Name] = string(content)
	}

	return entries
}

func readTarGzArchiveEntries(t *testing.T, archivePath string) map[string]string {
	t.Helper()

	//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open archive %q: %v", archivePath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("close archive %q: %v", archivePath, err)
		}
	}()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("open gzip stream %q: %v", archivePath, err)
	}
	defer func() {
		if err := gzipReader.Close(); err != nil {
			t.Fatalf("close gzip stream %q: %v", archivePath, err)
		}
	}()

	tarReader := tar.NewReader(gzipReader)
	entries := make(map[string]string)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read tar header in %q: %v", archivePath, err)
		}
		if header.FileInfo().IsDir() {
			continue
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar entry %q: %v", header.Name, err)
		}

		entries[header.Name] = string(content)
	}

	return entries
}

func readStagedFiles(t *testing.T, stagingDir string) map[string]string {
	t.Helper()

	entries := make(map[string]string)
	err := filepath.Walk(stagingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(stagingDir, path)
		if err != nil {
			return err
		}

		//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		entries[relPath] = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("read staging %q: %v", stagingDir, err)
	}

	return entries
}

func assertExactEntries(t *testing.T, got, want map[string]string) {
	t.Helper()

	gotKeys := slices.Sorted(maps.Keys(got))
	wantKeys := slices.Sorted(maps.Keys(want))
	if !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("unexpected entries: got %v, want %v", gotKeys, wantKeys)
	}

	for path, wantContent := range want {
		if got[path] != wantContent {
			t.Errorf("unexpected content for %q: got %q, want %q", path, got[path], wantContent)
		}
	}
}
