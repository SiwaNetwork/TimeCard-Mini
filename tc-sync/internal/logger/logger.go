// Package logger — единый вывод логов tc-sync с префиксом и учётом quiet.
package logger

import "log"

// Quiet при true отключает информационные сообщения (Info); Error выводится всегда.
var Quiet bool

// Info выводит сообщение с префиксом "tc-sync: ", если Quiet == false.
func Info(format string, args ...interface{}) {
	if Quiet {
		return
	}
	log.Printf("tc-sync: "+format, args...)
}

// Error выводит сообщение об ошибке с префиксом "tc-sync: " всегда.
func Error(format string, args ...interface{}) {
	log.Printf("tc-sync: "+format, args...)
}
