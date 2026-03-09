package cli

import (
	"errors"
)

// Коды выхода приложения
const (
	ExitSuccess   = 0
	ExitUsageErr  = 1 // Ошибки использования CLI (неверные флаги, неизвестные команды)
	ExitValError  = 2 // Ошибки бизнес-логики (непройденная валидация, сбой сборки)
	ExitConfigErr = 3 // Ошибки инфраструктуры/конфигурации (отсутствие файла настроек, ошибки прав доступа)
)

// CLIError — кастомная ошибка, содержащая код выхода
type CLIError struct {
	Code int
	Err  error
}

func (e *CLIError) Error() string {
	return e.Err.Error()
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// NewCLIError создает новую ошибку с кодом выхода
func NewCLIError(code int, err error) error {
	return &CLIError{Code: code, Err: err}
}

// GetExitCode возвращает код выхода для переданной ошибки
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}
	// По умолчанию возвращаем 1 для системных ошибок Cobra (флаги, аргументы)
	return ExitUsageErr
}
