package logger

// Logger provides structured logging with context
type Logger interface {
	Info(component, message string, fields map[string]interface{})
	Error(component string, err error, fields map[string]interface{})
	Warning(component, message string, fields map[string]interface{})
	Debug(component, message string, fields map[string]interface{})
}
