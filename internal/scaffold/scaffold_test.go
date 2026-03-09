package scaffold_test

import (
	"bx-pack/internal/cli"
	"bx-pack/internal/config"
	"bx-pack/internal/report"
	"os"
	"path/filepath"
	"testing"
)

func TestScaffold_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	reporter := report.NewReporter(report.TextFormat)

	// 1. Тест обычного запуска
	err := cli.Scaffold(reporter, false)
	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	// Проверяем наличие ключевых файлов
	expectedFiles := []string{
		config.DefaultConfigPath,
		"install/version.php",
		"install/index.php",
		"lang/ru/install/index.php",
		"lib/example.php",
		"admin/menu.php",
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// 2. Тест dry-run в другой временной директории
	tmpDirDry := t.TempDir()
	os.Chdir(tmpDirDry)

	err = cli.Scaffold(reporter, true)
	if err != nil {
		t.Fatalf("Scaffold dry-run failed: %v", err)
	}

	// В dry-run файлы не должны создаться
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("File %s should not exist in dry-run mode", f)
		}
	}

	// 3. Тест на неперезаписывание существующих файлов
	os.Chdir(tmpDir) // Возвращаемся в первую директорию
	existingContent := "CUSTOM CONTENT"
	customFile := "install/version.php"
	err = os.WriteFile(customFile, []byte(existingContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = cli.Scaffold(reporter, false)
	if err != nil {
		t.Fatalf("Scaffold failed on second run: %v", err)
	}

	data, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existingContent {
		t.Errorf("File %s was overwritten but should not have been", customFile)
	}
}

func TestScaffold_SuggestedID(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "vendor.mymodule")
	err := os.MkdirAll(moduleDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(moduleDir)

	reporter := report.NewReporter(report.TextFormat)
	err = cli.Scaffold(reporter, false)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedID := "vendor.mymodule"
	if cfg.Module.ID != expectedID {
		t.Errorf("Expected module ID %s, got %s", expectedID, cfg.Module.ID)
	}
}
