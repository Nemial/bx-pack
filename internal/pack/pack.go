package pack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bx-pack/internal/config"
)

type archiveFormat string

const (
	archiveFormatZIP   archiveFormat = "zip"
	archiveFormatTarGz archiveFormat = "tar.gz"
)

// IsExcluded проверяет, должен ли путь быть исключен на основе списка паттернов.
func IsExcluded(relPath string, exclude []string) (bool, error) {
	for _, pattern := range exclude {
		if pattern == "" {
			continue
		}

		if relPath == pattern || strings.HasPrefix(relPath, pattern+string(filepath.Separator)) {
			return true, nil
		}

		match, err := filepath.Match(pattern, relPath)
		if err != nil {
			return false, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		if match {
			return true, nil
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[:i], string(filepath.Separator))
			match, err = filepath.Match(pattern, parent)
			if err == nil && match {
				return true, nil
			}
		}

		match, err = filepath.Match(pattern, filepath.Base(relPath))
		if err == nil && match {
			return true, nil
		}
	}
	return false, nil
}

func PrepareStaging(cfg config.Config) error {
	if err := os.RemoveAll(cfg.Build.StagingDir); err != nil {
		return fmt.Errorf("cleanup staging dir: %w", err)
	}

	if err := os.MkdirAll(cfg.Build.StagingDir, 0o750); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	absStaging := cfg.Build.StagingDir
	absOutput := cfg.Build.OutputDir

	return filepath.Walk(cfg.Build.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == cfg.Build.SourceDir {
			return nil
		}

		relPath, err := filepath.Rel(cfg.Build.SourceDir, path)
		if err != nil {
			return err
		}

		excluded, err := IsExcluded(relPath, cfg.Exclude)
		if err != nil {
			return err
		}
		if excluded {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

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
	return filepath.Join(cfg.Build.OutputDir, resolveArchiveName(cfg))
}

func CreateArchive(cfg config.Config) (string, error) {
	if err := os.MkdirAll(cfg.Build.OutputDir, 0o750); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	archiveName := resolveArchiveName(cfg)
	format, err := detectArchiveFormat(archiveName)
	if err != nil {
		return "", err
	}

	archivePath := filepath.Join(cfg.Build.OutputDir, archiveName)
	baseDirName := archiveBaseName(archiveName, format)

	//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
	outFile, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("create archive file: %w", err)
	}
	defer outFile.Close()

	switch format {
	case archiveFormatZIP:
		zipWriter := zip.NewWriter(outFile)
		if err := writeZIPArchive(zipWriter, cfg.Build.StagingDir, baseDirName); err != nil {
			_ = zipWriter.Close()
			return "", fmt.Errorf("zip staging: %w", err)
		}
		if err := zipWriter.Close(); err != nil {
			return "", fmt.Errorf("finalize zip archive: %w", err)
		}
	case archiveFormatTarGz:
		gzipWriter := gzip.NewWriter(outFile)
		tarWriter := tar.NewWriter(gzipWriter)

		if err := writeTarArchive(tarWriter, cfg.Build.StagingDir, baseDirName); err != nil {
			_ = tarWriter.Close()
			_ = gzipWriter.Close()
			return "", fmt.Errorf("tar.gz staging: %w", err)
		}
		if err := tarWriter.Close(); err != nil {
			_ = gzipWriter.Close()
			return "", fmt.Errorf("finalize tar stream: %w", err)
		}
		if err := gzipWriter.Close(); err != nil {
			return "", fmt.Errorf("finalize gzip stream: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported archive format %q", format)
	}

	return archivePath, nil
}

func resolveArchiveName(cfg config.Config) string {
	archiveName := cfg.Build.ArchiveName
	archiveName = strings.ReplaceAll(archiveName, "{module.id}", cfg.Module.ID)
	archiveName = strings.ReplaceAll(archiveName, "{module.version}", cfg.Module.Version)
	return archiveName
}

func detectArchiveFormat(archiveName string) (archiveFormat, error) {
	switch {
	case strings.HasSuffix(archiveName, ".tar.gz"):
		return archiveFormatTarGz, nil
	case strings.HasSuffix(archiveName, ".zip"):
		return archiveFormatZIP, nil
	default:
		return "", fmt.Errorf("unsupported archive format %q: expected .zip or .tar.gz", archiveName)
	}
}

func archiveBaseName(archiveName string, format archiveFormat) string {
	switch format {
	case archiveFormatZIP:
		return strings.TrimSuffix(archiveName, ".zip")
	case archiveFormatTarGz:
		return strings.TrimSuffix(archiveName, ".tar.gz")
	default:
		return archiveName
	}
}

func walkStagingFiles(stagingDir string, visit func(path, relPath string, info os.FileInfo) error) error {
	return filepath.Walk(stagingDir, func(path string, info os.FileInfo, err error) error {
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

		return visit(path, relPath, info)
	})
}

func writeZIPArchive(w *zip.Writer, stagingDir, baseDirName string) error {
	return walkStagingFiles(stagingDir, func(path, relPath string, info os.FileInfo) error {
		archivePath := filepath.ToSlash(filepath.Join(baseDirName, relPath))

		f, err := w.Create(archivePath)
		if err != nil {
			return err
		}

		//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(f, in)
		return err
	})
}

func writeTarArchive(w *tar.Writer, stagingDir, baseDirName string) error {
	return walkStagingFiles(stagingDir, func(path, relPath string, info os.FileInfo) error {
		archivePath := filepath.ToSlash(filepath.Join(baseDirName, relPath))

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = archivePath

		if err := w.WriteHeader(header); err != nil {
			return err
		}

		//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(w, in)
		return err
	})
}

func copyFile(src, dst string) error {
	//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
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
