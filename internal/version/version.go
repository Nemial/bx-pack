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

func BumpVersion(path string, bumpLevel string) (oldVersion, newVersion string, err error) {
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
			parts := strings.Split(oldVer, ".")
			if len(parts) != 3 {
				return "", "", fmt.Errorf("неверный формат SemVer %q в %q", oldVer, path)
			}
			major, err := strconv.Atoi(parts[0])
			if err != nil {
				return "", "", fmt.Errorf("неверный major в версии %q: %w", oldVer, err)
			}
			minor, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", "", fmt.Errorf("неверный minor в версии %q: %w", oldVer, err)
			}
			patch, err := strconv.Atoi(parts[2])
			if err != nil {
				return "", "", fmt.Errorf("неверный patch в версии %q: %w", oldVer, err)
			}
			switch bumpLevel {
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
				return "", "", fmt.Errorf("неверный уровень bump %q, ожидаются patch/minor/major", bumpLevel)
			}
			newVer = fmt.Sprintf("%d.%d.%d", major, minor, patch)
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
