package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"bx-pack/internal/config"
	"bx-pack/internal/report"
)

//go:embed templates/*
var templatesFS embed.FS

const (
	defaultVendor = "myvendor"
	defaultModule = "module"
)

type templateData struct {
	ModuleID       string
	ClassName      string
	ClassNameUpper string
	Namespace      string
	VersionDate    string
}

// Run выполняет генерацию базовой структуры модуля.
func Run(reporter report.Reporter, dryRun bool) error {
	reporter.SetCommand("scaffold")

	// Определяем ID модуля (пытаемся взять из текущей папки или дефолт)
	moduleID := getSuggestedModuleID()
	data := prepareTemplateData(moduleID)

	filesToGenerate := map[string]string{
		"install/version.php":       "templates/version.php.tmpl",
		"install/index.php":         "templates/index.php.tmpl",
		"lang/ru/install/index.php": "templates/lang_index.php.tmpl",
		"lib/example.php":           "templates/example_lib.php.tmpl",
	}

	files := map[string]string{
		"admin/menu.php":     "<?php\n// Меню административной панели\nreturn [];\n",
		"include.php":        "<?php\n// Подключение автозагрузки или констант модуля\n",
		"prolog.php":         "<?php\n// Пре-инициализация\ndefine('ADMIN_MODULE_NAME', '" + moduleID + "');\n",
		"default_option.php": "<?php\n$ " + strings.ReplaceAll(moduleID, ".", "_") + "_default_option = [\n];\n",
		"options.php":        "<?php\n// Настройки модуля в административной панели\n",
	}

	// Генерируем конфиг через config пакет
	configContent, err := config.GenerateForModuleID(moduleID)
	if err != nil {
		return fmt.Errorf("ошибка генерации конфига: %w", err)
	}
	files[config.DefaultConfigPath] = configContent

	// Рендерим остальные шаблоны
	for path, tmplPath := range filesToGenerate {
		content, err := renderTemplate(tmplPath, data)
		if err != nil {
			return fmt.Errorf("ошибка рендеринга шаблона %s: %w", tmplPath, err)
		}
		files[path] = content
	}

	dirs := []string{
		"install/components",
		"install/templates",
		"install/db",
		"assets/css",
		"assets/js",
	}

	if dryRun {
		reporter.PrintInfo("Режим dry-run: файлы не будут созданы.")
	}

	for _, dir := range dirs {
		if dryRun {
			reporter.PrintInfo(fmt.Sprintf("Будет создана директория: %s", dir))
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("ошибка создания директории %s: %w", dir, err)
		}
	}

	// Сортируем ключи для детерминированного вывода (опционально, но полезно для тестов)
	// Но здесь map, порядок не гарантирован.

	for path, content := range files {
		if dryRun {
			reporter.PrintInfo(fmt.Sprintf("Будет создан файл: %s", path))
			continue
		}

		// Проверяем существование
		if _, err := os.Stat(path); err == nil {
			reporter.PrintInfo(fmt.Sprintf("Файл %s уже существует, пропуск", path))
			continue
		}

		// Создаем поддиректории для файла
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("ошибка создания директории для %s: %w", path, err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("ошибка записи файла %s: %w", path, err)
		}
		reporter.PrintInfo(fmt.Sprintf("Создан файл: %s", path))
	}

	if !dryRun {
		reporter.PrintSuccess("Базовая структура модуля успешно создана!")
		reporter.PrintInfo("Теперь вы можете настроить .bxpack.yml и запустить 'bx-pack validate'")
	}

	return nil
}

func getSuggestedModuleID() string {
	wd, err := os.Getwd()
	if err != nil {
		return defaultVendor + "." + defaultModule
	}
	base := filepath.Base(wd)
	if strings.Contains(base, ".") {
		return base
	}
	return defaultVendor + "." + base
}

func prepareTemplateData(moduleID string) templateData {
	className := strings.ReplaceAll(moduleID, ".", "_")
	parts := strings.Split(moduleID, ".")
	var namespaceParts []string
	for _, p := range parts {
		namespaceParts = append(namespaceParts, strings.Title(p))
	}

	return templateData{
		ModuleID:       moduleID,
		ClassName:      className,
		ClassNameUpper: strings.ToUpper(className),
		Namespace:      strings.Join(namespaceParts, "\\"),
		VersionDate:    time.Now().Format("2006-01-02 15:04:05"),
	}
}

func renderTemplate(tmplPath string, data templateData) (string, error) {
	tmplContent, err := templatesFS.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(tmplPath).Parse(string(tmplContent))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
