package pack

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bx-pack/internal/config"
)

func PrepareStaging(cfg config.Config) error {
	// Нормализуем пути перед работой
	if err := cfg.NormalizePaths(); err != nil {
		return fmt.Errorf("normalize paths: %w", err)
	}

	// Очистка и создание staging директории
	if err := os.RemoveAll(cfg.Build.StagingDir); err != nil {
		return fmt.Errorf("cleanup staging dir: %w", err)
	}

	if err := os.MkdirAll(cfg.Build.StagingDir, 0755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	absStaging := cfg.Build.StagingDir
	absOutput := cfg.Build.OutputDir

	// Копирование файлов
	return filepath.Walk(cfg.Build.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == cfg.Build.SourceDir {
			return nil
		}

		// Получаем относительный путь от sourceDir
		relPath, err := filepath.Rel(cfg.Build.SourceDir, path)
		if err != nil {
			return err
		}

		// Простая логика исключений
		for _, exc := range cfg.Exclude {
			if relPath == exc || strings.HasPrefix(relPath, exc+string(filepath.Separator)) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Исключаем саму staging и output директории
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(absPath, absStaging) || strings.HasPrefix(absPath, absOutput) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(cfg.Build.StagingDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

func GetArchivePath(cfg config.Config) string {
	archiveName := cfg.Build.ArchiveName
	archiveName = strings.ReplaceAll(archiveName, "{module.id}", cfg.Module.ID)
	archiveName = strings.ReplaceAll(archiveName, "{module.version}", cfg.Module.Version)

	return filepath.Join(cfg.Build.OutputDir, archiveName)
}

func CreateArchive(cfg config.Config) (string, error) {
	// Нормализуем пути перед работой
	if err := cfg.NormalizePaths(); err != nil {
		return "", fmt.Errorf("normalize paths: %w", err)
	}

	if err := os.MkdirAll(cfg.Build.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	archivePath := GetArchivePath(cfg)

	outFile, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("create archive file: %w", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	err = filepath.Walk(cfg.Build.StagingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(cfg.Build.StagingDir, path)
		if err != nil {
			return err
		}

		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(f, in)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("zip staging: %w", err)
	}

	return archivePath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}
