package pack

import (
	"archive/zip"
	"bx-pack/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPipeline(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Setup source directory with files
	sourceDir := filepath.Join(tempDir, "source")
	err := os.MkdirAll(sourceDir, 0755)
	if err != nil {
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

	for path, content := range files {
		fullPath := filepath.Join(sourceDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 2. Setup config
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

	// 3. Prepare Staging
	err = PrepareStaging(cfg)
	if err != nil {
		t.Fatalf("PrepareStaging failed: %v", err)
	}

	// 4. Verify Staging
	expectedStaged := []string{
		"include.txt",
		"nested/include2.txt",
		"exclude.txt.bak",
	}
	unexpectedStaged := []string{
		"exclude.txt",
		"excluded/some.txt",
		".git/config",
		"nested/excluded_in_nested",
	}

	for _, rel := range expectedStaged {
		stagedPath := filepath.Join(cfg.Build.StagingDir, rel)
		if _, err := os.Stat(stagedPath); os.IsNotExist(err) {
			t.Errorf("expected file %s to be staged, but it's missing", rel)
		}
	}

	for _, rel := range unexpectedStaged {
		stagedPath := filepath.Join(cfg.Build.StagingDir, rel)
		if _, err := os.Stat(stagedPath); !os.IsNotExist(err) {
			t.Errorf("expected file %s NOT to be staged, but it's present", rel)
		}
	}

	// 5. Create Archive
	archivePath, err := CreateArchive(cfg)
	if err != nil {
		t.Fatalf("CreateArchive failed: %v", err)
	}

	expectedArchiveName := "test.module-1.2.3.zip"
	if filepath.Base(archivePath) != expectedArchiveName {
		t.Errorf("expected archive name %s, got %s", expectedArchiveName, filepath.Base(archivePath))
	}

	// 6. Verify Archive Content
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("failed to open archive: %v", err)
	}
	defer r.Close()

	archiveFiles := make(map[string]bool)
	for _, f := range r.File {
		archiveFiles[f.Name] = true
	}

	for _, rel := range expectedStaged {
		if !archiveFiles[rel] {
			t.Errorf("expected file %s to be in archive, but it's missing", rel)
		}
	}

	for _, rel := range unexpectedStaged {
		if archiveFiles[rel] {
			t.Errorf("expected file %s NOT to be in archive, but it's present", rel)
		}
	}
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
