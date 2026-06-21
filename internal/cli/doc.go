// Package cli реализует точки входа CLI-команд bx-pack.
//
// Каждая команда (init, scaffold, validate, build, version) представлена
// отдельной функцией в commands.go, которая оркестрирует работу config,
// validate, pack, scaffold и version пакетов, делегируя вывод report.
package cli
