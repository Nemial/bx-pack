package pack

import (
	"archive/zip"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"bx-pack/internal/config"
)

func writeTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("create parent dir for %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write file %q: %v", path, err)
		}
	}
}

func readArchiveEntries(t *testing.T, archivePath string) map[string]string {
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

func TestBuildPipeline(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"include.txt":               "content of include.txt",
		"nested/include2.txt":       "content of nested/include2.txt",
		"exclude.txt":               "content of exclude.txt",
		"excluded/some.txt":         "content of excluded/some.txt",
		".git/config":               "git config content",
		"nested/excluded_in_nested": "should be excluded",
		"exclude.txt.bak":           "should NOT be excluded",
	}
	writeTestFiles(t, sourceDir, files)

	cfg := config.Default()
	cfg.Module.ID = "test.module"
	cfg.Module.Version = "1.2.3"
	cfg.Build.SourceDir = sourceDir
	cfg.Build.OutputDir = filepath.Join(tempDir, "dist")
	cfg.Build.StagingDir = filepath.Join(tempDir, ".bxpack/staging")
	cfg.Exclude = []string{
		"exclude.txt",
		"excluded",
		".git",
		"nested/excluded_in_nested",
	}

	if err := PrepareStaging(cfg); err != nil {
		t.Fatalf("PrepareStaging failed: %v", err)
	}

	expectedEntries := map[string]string{
		"include.txt":         files["include.txt"],
		"nested/include2.txt": files["nested/include2.txt"],
		"exclude.txt.bak":     files["exclude.txt.bak"],
	}
	assertExactEntries(t, readStagedFiles(t, cfg.Build.StagingDir), expectedEntries)

	archivePath, err := CreateArchive(cfg)
	if err != nil {
		t.Fatalf("CreateArchive failed: %v", err)
	}

	expectedArchiveName := "test.module-1.2.3.zip"
	if filepath.Base(archivePath) != expectedArchiveName {
		t.Errorf("expected archive name %s, got %s", expectedArchiveName, filepath.Base(archivePath))
	}

	expectedArchiveEntries := make(map[string]string)
	baseDir := "test.module-1.2.3"
	for k, v := range expectedEntries {
		expectedArchiveEntries[baseDir+"/"+k] = v
	}
	assertExactEntries(t, readArchiveEntries(t, archivePath), expectedArchiveEntries)
}

func TestPrepareStaging_Errors(t *testing.T) {
	t.Run("source dir not found", func(t *testing.T) {
		cfg := config.Default()
		cfg.Build.SourceDir = "/non-existent-path-12345"
		cfg.Build.StagingDir = t.TempDir()

		err := PrepareStaging(cfg)
		if err == nil {
			t.Error("expected error for missing source dir, got nil")
		}
	})

	t.Run("staging dir creation failure", func(t *testing.T) {
		// Create a file where the staging directory should be
		tmpDir := t.TempDir()
		stagingPath := filepath.Join(tmpDir, "staging-file")
		if err := os.WriteFile(stagingPath, []byte("not a dir"), 0644); err != nil {
			t.Fatal(err)
		}

		// Try to use this file path as staging dir (MkdirAll should fail)
		// Wait, MkdirAll might not fail if it's already there, but if we need to create subdirs it will.
		cfg := config.Default()
		cfg.Build.StagingDir = filepath.Join(stagingPath, "subdir")

		err := PrepareStaging(cfg)
		if err == nil {
			t.Error("expected error for invalid staging dir path, got nil")
		}
	})
}

func TestPathSafety(t *testing.T) {
	t.Run("exclude staging and output even if inside source", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(sourceDir, 0o755); err != nil {
			t.Fatal(err)
		}

		stagingDir := filepath.Join(sourceDir, ".staging")
		outputDir := filepath.Join(sourceDir, "dist")

		writeTestFiles(t, sourceDir, map[string]string{"keep.txt": "keep"})
		writeTestFiles(t, stagingDir, map[string]string{"skip-staging.txt": "skip"})
		writeTestFiles(t, outputDir, map[string]string{"skip-output.txt": "skip"})

		cfg := config.Default()
		cfg.Build.SourceDir = sourceDir
		cfg.Build.StagingDir = stagingDir
		cfg.Build.OutputDir = outputDir

		if err := PrepareStaging(cfg); err != nil {
			t.Fatalf("PrepareStaging failed: %v", err)
		}

		assertExactEntries(t, readStagedFiles(t, cfg.Build.StagingDir), map[string]string{"keep.txt": "keep"})

		archivePath, err := CreateArchive(cfg)
		if err != nil {
			t.Fatalf("CreateArchive failed: %v", err)
		}

		expectedArchiveEntries := map[string]string{
			"example.module-/keep.txt": "keep",
		}
		assertExactEntries(t, readArchiveEntries(t, archivePath), expectedArchiveEntries)
	})

	t.Run("exclude internal directories if inside source", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(sourceDir, 0o755); err != nil {
			t.Fatal(err)
		}

		stagingDir := filepath.Join(sourceDir, ".bxpack/staging")
		outputDir := filepath.Join(sourceDir, "dist")

		writeTestFiles(t, sourceDir, map[string]string{"root.txt": "root"})
		writeTestFiles(t, stagingDir, map[string]string{"staged.txt": "staged"})
		writeTestFiles(t, outputDir, map[string]string{"archived.zip": "zip"})

		cfg := config.Default()
		cfg.Build.SourceDir = sourceDir
		cfg.Build.StagingDir = stagingDir
		cfg.Build.OutputDir = outputDir

		if err := PrepareStaging(cfg); err != nil {
			t.Fatalf("PrepareStaging failed: %v", err)
		}

		assertExactEntries(t, readStagedFiles(t, cfg.Build.StagingDir), map[string]string{"root.txt": "root"})

		archivePath, err := CreateArchive(cfg)
		if err != nil {
			t.Fatalf("CreateArchive failed: %v", err)
		}

		assertExactEntries(t, readArchiveEntries(t, archivePath), map[string]string{"example.module-/root.txt": "root"})
	})

	t.Run("exclude nested directories recursively", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceDir := filepath.Join(tempDir, "src")
		writeTestFiles(t, sourceDir, map[string]string{
			"a/b/c/keep.txt":        "keep",
			"a/b/c/keep-too.txt":    "keep-too",
			"a/b/excluded/skip.txt": "skip",
			"a/b/excluded/deep.txt": "deep-skip",
			"a/excluded.txt":        "should stay",
			"a/b/excluded-file.txt": "should stay too",
		})

		cfg := config.Default()
		cfg.Build.SourceDir = sourceDir
		cfg.Build.StagingDir = filepath.Join(tempDir, "staging")
		cfg.Build.OutputDir = filepath.Join(tempDir, "dist")
		cfg.Exclude = []string{"a/b/excluded"}

		if err := PrepareStaging(cfg); err != nil {
			t.Fatal(err)
		}

		expectedEntries := map[string]string{
			"a/b/c/keep.txt":        "keep",
			"a/b/c/keep-too.txt":    "keep-too",
			"a/excluded.txt":        "should stay",
			"a/b/excluded-file.txt": "should stay too",
		}
		assertExactEntries(t, readStagedFiles(t, cfg.Build.StagingDir), expectedEntries)

		archivePath, err := CreateArchive(cfg)
		if err != nil {
			t.Fatalf("CreateArchive failed: %v", err)
		}

		expectedArchiveEntries := make(map[string]string)
		for k, v := range expectedEntries {
			expectedArchiveEntries["example.module-/"+k] = v
		}
		assertExactEntries(t, readArchiveEntries(t, archivePath), expectedArchiveEntries)
	})

	t.Run("relative paths normalization", func(t *testing.T) {
		tempDir := t.TempDir()
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("get working dir: %v", err)
		}
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("change working dir to temp dir: %v", err)
		}
		defer func() {
			if err := os.Chdir(oldWd); err != nil {
				t.Fatalf("restore working dir: %v", err)
			}
		}()

		writeTestFiles(t, filepath.Join(tempDir, "src"), map[string]string{"test.txt": "test"})

		cfg := config.Default()
		cfg.Build.SourceDir = "./src"
		cfg.Build.StagingDir = "./.staging"
		cfg.Build.OutputDir = "./dist"

		if err := PrepareStaging(cfg); err != nil {
			t.Fatalf("PrepareStaging failed: %v", err)
		}

		assertExactEntries(t, readStagedFiles(t, filepath.Join(tempDir, ".staging")), map[string]string{"test.txt": "test"})
	})
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		exclude  []string
		expected bool
	}{
		{
			name:     "exact file match",
			relPath:  "config.php",
			exclude:  []string{"config.php"},
			expected: true,
		},
		{
			name:     "exact dir match",
			relPath:  "tests",
			exclude:  []string{"tests"},
			expected: true,
		},
		{
			name:     "file in excluded dir",
			relPath:  "tests/unit/test.php",
			exclude:  []string{"tests"},
			expected: true,
		},
		{
			name:     "glob extension match",
			relPath:  "error.log",
			exclude:  []string{"*.log"},
			expected: true,
		},
		{
			name:     "glob extension match in subdir",
			relPath:  "logs/error.log",
			exclude:  []string{"*.log"},
			expected: true,
		},
		{
			name:     "glob subdir match",
			relPath:  "temp/cache/file.txt",
			exclude:  []string{"temp/*"},
			expected: true,
		},
		{
			name:     "no match",
			relPath:  "install/index.php",
			exclude:  []string{"tests", "*.log"},
			expected: false,
		},
		{
			name:     "empty pattern",
			relPath:  "install/index.php",
			exclude:  []string{""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsExcluded(tt.relPath, tt.exclude)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("IsExcluded(%q, %v) = %v, want %v", tt.relPath, tt.exclude, got, tt.expected)
			}
		})
	}
}

func TestPrepareStaging_GlobExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	stagingDir := filepath.Join(tmpDir, "staging")
	outputDir := filepath.Join(tmpDir, "dist")

	files := map[string]string{
		"install/index.php":   "php content",
		"tests/test.php":      "test content",
		"debug.log":           "log content",
		"logs/app.log":        "app log",
		"temp/cache/item.txt": "cache item",
		"readme.txt":          "readme",
	}
	// Создаем директории вручную, так как writeTestFiles может не создавать их для пустых путей (хотя здесь пути с файлами)
	os.MkdirAll(filepath.Join(sourceDir, "install"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "tests"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "logs"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "temp/cache"), 0755)

	writeTestFiles(t, sourceDir, files)

	cfg := config.Config{
		Module: config.Module{ID: "test.module", Version: "1.0.0"},
		Build: config.Build{
			SourceDir:  sourceDir,
			StagingDir: stagingDir,
			OutputDir:  outputDir,
		},
		Exclude: []string{
			"tests",
			"*.log",
			"temp/*",
		},
	}

	err := PrepareStaging(cfg)
	if err != nil {
		t.Fatalf("PrepareStaging failed: %v", err)
	}

	staged := readStagedFiles(t, stagingDir)

	expected := map[string]string{
		"install/index.php": "php content",
		"readme.txt":        "readme",
	}

	assertExactEntries(t, staged, expected)
}
