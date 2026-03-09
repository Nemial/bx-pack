package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bx-pack/internal/config"
	"bx-pack/internal/report"
)

const (
	defaultVendor = "myvendor"
	defaultModule = "module"
)

// Run выполняет генерацию базовой структуры модуля.
func Run(reporter report.Reporter, dryRun bool) error {
	reporter.SetCommand("scaffold")

	// Определяем ID модуля (пытаемся взять из текущей папки или дефолт)
	moduleID := getSuggestedModuleID()

	files := map[string]string{
		config.DefaultConfigPath:    generateConfigTemplate(moduleID),
		"install/version.php":       generateVersionPHP(),
		"install/index.php":         generateIndexPHP(moduleID),
		"lang/ru/install/index.php": generateLangIndexPHP(moduleID),
		"lib/example.php":           generateExampleLib(moduleID),
		"admin/menu.php":            "<?php\n// Меню административной панели\nreturn [];\n",
		"include.php":               "<?php\n// Подключение автозагрузки или констант модуля\n",
		"prolog.php":                "<?php\n// Пре-инициализация\ndefine('ADMIN_MODULE_NAME', '" + moduleID + "');\n",
		"default_option.php":        "<?php\n$ " + strings.ReplaceAll(moduleID, ".", "_") + "_default_option = [\n];\n",
		"options.php":               "<?php\n// Настройки модуля в административной панели\n",
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

func generateConfigTemplate(moduleID string) string {
	tmpl := config.GenerateTemplate()
	// Заменяем дефолтный ID в шаблоне на предложенный
	return strings.Replace(tmpl, `id: "myvendor.my-module"`, fmt.Sprintf(`id: %q`, moduleID), 1)
}

func generateVersionPHP() string {
	now := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf(`<?php
$arModuleVersion = [
    "VERSION" => "0.1.0",
    "VERSION_DATE" => "%s",
];
`, now)
}

func generateIndexPHP(moduleID string) string {
	className := strings.ReplaceAll(moduleID, ".", "_")
	langName := "GetMessage('" + strings.ToUpper(className) + "_MODULE_NAME')"
	langDesc := "GetMessage('" + strings.ToUpper(className) + "_MODULE_DESC')"

	return fmt.Sprintf(`<?php
use Bitrix\Main\Localization\Loc;
use Bitrix\Main\ModuleManager;

Loc::loadMessages(__FILE__);

class %s extends CModule
{
    public $MODULE_ID = '%s';
    public $MODULE_VERSION;
    public $MODULE_VERSION_DATE;
    public $MODULE_NAME;
    public $MODULE_DESCRIPTION;

    public function __construct()
    {
        $arModuleVersion = [];
        include(__DIR__ . "/version.php");

        $this->MODULE_VERSION = $arModuleVersion["VERSION"];
        $this->MODULE_VERSION_DATE = $arModuleVersion["VERSION_DATE"];
        $this->MODULE_NAME = %s;
        $this->MODULE_DESCRIPTION = %s;

        $this->PARTNER_NAME = "My Vendor";
        $this->PARTNER_URI = "https://example.com";
    }

    public function DoInstall()
    {
        ModuleManager::registerModule($this->MODULE_ID);
    }

    public function DoUninstall()
    {
        ModuleManager::unRegisterModule($this->MODULE_ID);
    }
}
`, className, moduleID, langName, langDesc)
}

func generateLangIndexPHP(moduleID string) string {
	className := strings.ToUpper(strings.ReplaceAll(moduleID, ".", "_"))
	return fmt.Sprintf(`<?php
$MESS["%s_MODULE_NAME"] = "Название модуля %s";
$MESS["%s_MODULE_DESC"] = "Описание модуля %s";
`, className, moduleID, className, moduleID)
}

func generateExampleLib(moduleID string) string {
	parts := strings.Split(moduleID, ".")
	namespace := ""
	for _, p := range parts {
		namespace += strings.Title(p) + "\\"
	}
	namespace = strings.TrimSuffix(namespace, "\\")

	return fmt.Sprintf(`<?php
namespace %s;

class Example
{
    public static function doSomething()
    {
        return "Hello from %s";
    }
}
`, namespace, moduleID)
}
