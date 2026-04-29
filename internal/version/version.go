package version

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	versionDoubleRE = regexp.MustCompile(`(.*?)\s*(?:\$VERSION|"VERSION")\s*(?:=|=>)\s*"\s*([^"]*)\s*"\s*(?:;|,)?\s*`)
	versionSingleRE = regexp.MustCompile(`(.*?)\s*(?:\$VERSION|'VERSION')\s*(?:=|=>)\s*'\s*([^']*)\s*'\s*(?:;|,)?\s*`)
	dateDoubleRE    = regexp.MustCompile(`(.*?)\s*(?:\$VERSION_DATE|"VERSION_DATE")\s*(?:=|=>)\s*"\s*([^"]*)\s*"\s*(?:;|,)?\s*`)
	dateSingleRE    = regexp.MustCompile(`(.*?)\s*(?:\$VERSION_DATE|'VERSION_DATE')\s*(?:=|=>)\s*'\s*([^']*)\s*'\s*(?:;|,)?\s*`)
)

const (
	SchemeSemVer     = "semver"
	SchemeCalVer     = "calver"
	SchemeYearSemVer = "year-semver"
	SchemeCustom     = "custom"
)

func parseAssign(line, varName string) (prefix, operator, quote, value string, ok bool) {
	if varName == "VERSION" {
		for _, re := range []*regexp.Regexp{versionDoubleRE, versionSingleRE} {
			match := re.FindStringSubmatch(line)
			if len(match) == 3 {
				q := "\""
				if re == versionSingleRE {
					q = "'"
				}
				op := "="
				if strings.Contains(match[0], "=>") {
					op = "=>"
				}
				var actualVarName string
				if strings.Contains(match[0], "$VERSION") {
					actualVarName = "$VERSION"
				} else {
					actualVarName = q + "VERSION" + q
				}
				return match[1], actualVarName + " " + op, q, strings.TrimSpace(match[2]), true
			}
		}
	} else if varName == "VERSION_DATE" {
		for _, re := range []*regexp.Regexp{dateDoubleRE, dateSingleRE} {
			match := re.FindStringSubmatch(line)
			if len(match) == 3 {
				q := "\""
				if re == dateSingleRE {
					q = "'"
				}
				op := "="
				if strings.Contains(match[0], "=>") {
					op = "=>"
				}
				var actualVarName string
				if strings.Contains(match[0], "$VERSION_DATE") {
					actualVarName = "$VERSION_DATE"
				} else {
					actualVarName = q + "VERSION_DATE" + q
				}
				return match[1], actualVarName + " " + op, q, strings.TrimSpace(match[2]), true
			}
		}
	}
	return "", "", "", "", false
}

func ParseVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("чтение файла %q: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		_, _, _, value, ok := parseAssign(line, "VERSION")
		if ok {
			return value, nil
		}
	}
	return "", fmt.Errorf("строка с $VERSION не найдена в %q", path)
}

func BumpVersion(path string, scheme string, bumpLevel string) (oldVersion, newVersion string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("чтение файла %q: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	var versionUpdated bool
	bumpLevel = strings.ToLower(bumpLevel)

	var oldVer, newVer string

	// Парсим и обновляем VERSION
	for i, line := range lines {
		prefix, operator, quote, val, ok := parseAssign(line, "VERSION")
		if ok {
			oldVer = val
			newVer, err = calculateNewVersion(oldVer, scheme, bumpLevel)
			if err != nil {
				return "", "", err
			}

			suffix := ";"
			if strings.Contains(operator, "=>") {
				suffix = ","
			}
			lines[i] = prefix + operator + " " + quote + newVer + quote + suffix
			versionUpdated = true
			break
		}
	}

	// Обновляем VERSION_DATE
	newDate := time.Now().Format("2006-01-02 15:04:05")
	for i, line := range lines {
		prefix, operator, quote, _, ok := parseAssign(line, "VERSION_DATE")
		if ok {
			suffix := ";"
			if strings.Contains(operator, "=>") {
				suffix = ","
			}
			lines[i] = prefix + operator + " " + quote + newDate + quote + suffix
			break
		}
	}

	if !versionUpdated {
		return "", "", fmt.Errorf("строка $VERSION не найдена в %q", path)
	}

	newData := strings.Join(lines, "\n")
	if !strings.HasSuffix(newData, "\n") {
		newData += "\n"
	}
	if err := os.WriteFile(path, []byte(newData), 0644); err != nil {
		return "", "", fmt.Errorf("запись файла %q: %w", path, err)
	}
	return oldVer, newVer, nil
}

func calculateNewVersion(oldVer, scheme, level string) (string, error) {
	now := time.Now()
	if level == "auto" {
		switch scheme {
		case SchemeSemVer:
			level = "patch"
		case SchemeCalVer:
			level = "patch"
		case SchemeYearSemVer:
			level = "patch"
		}
	}

	switch scheme {
	case SchemeSemVer:
		return bumpSemVer(oldVer, level)
	case SchemeCalVer:
		if level != "patch" {
			return "", fmt.Errorf("схема %q поддерживает только bump patch или auto", scheme)
		}
		return bumpCalVer(oldVer, now)
	case SchemeYearSemVer:
		if level == "major" {
			return "", fmt.Errorf("схема %q не поддерживает bump major", scheme)
		}
		return bumpYearSemVer(oldVer, level, now)
	case SchemeCustom:
		return "", fmt.Errorf("автоматический bump не поддерживается для схемы %q", scheme)
	default:
		return "", fmt.Errorf("неизвестная схема версионирования: %q", scheme)
	}
}

func bumpSemVer(oldVer, level string) (string, error) {
	parts := strings.Split(oldVer, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("неверный формат SemVer %q", oldVer)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("неверный major в версии %q: %w", oldVer, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("неверный minor в версии %q: %w", oldVer, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("неверный patch в версии %q: %w", oldVer, err)
	}

	switch level {
	case "patch":
		patch++
	case "minor":
		minor++
		patch = 0
	case "major":
		major++
		minor = 0
		patch = 0
	default:
		return "", fmt.Errorf("неверный уровень bump %q для semver, ожидаются patch/minor/major", level)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func bumpCalVer(oldVer string, now time.Time) (string, error) {
	parts := strings.Split(oldVer, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("неверный формат CalVer %q, ожидается YYYY.M.PATCH", oldVer)
	}

	year := now.Year()
	month := int(now.Month())

	oldYear, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("неверный год в версии %q: %w", oldVer, err)
	}
	oldMonth, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("неверный месяц в версии %q: %w", oldVer, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("неверный patch в версии %q: %w", oldVer, err)
	}

	if year == oldYear && month == oldMonth {
		patch++
	} else {
		patch = 0
	}

	return fmt.Sprintf("%d.%d.%d", year, month, patch), nil
}

func bumpYearSemVer(oldVer string, level string, now time.Time) (string, error) {
	parts := strings.Split(oldVer, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("неверный формат Year-SemVer %q, ожидается YYYY.MINOR.PATCH", oldVer)
	}

	year := now.Year()
	oldYear, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("неверный год в версии %q: %w", oldVer, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("неверный minor в версии %q: %w", oldVer, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("неверный patch в версии %q: %w", oldVer, err)
	}

	if year != oldYear {
		return fmt.Sprintf("%d.1.0", year), nil
	}

	switch level {
	case "patch":
		patch++
	case "minor":
		minor++
		patch = 0
	default:
		return "", fmt.Errorf("неверный уровень bump %q для year-semver, ожидаются patch/minor", level)
	}

	return fmt.Sprintf("%d.%d.%d", year, minor, patch), nil
}
