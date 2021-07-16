package log

type Config struct {
	Dir      string
	FileSize int64
	FileNum  int32
	Env      string
	Level    string
	FileName string
}

func Init(c *Config) {
	newLogger(c)
	l.run()
}

func Close() {
	l.stop()
}

func Debug(traceId string, args ...interface{}) {
	l.p(traceId, DEBUG, args...)
}

func Debugf(traceId string, format string, args ...interface{}) {
	l.pf(traceId, DEBUG, format, args...)
}

func Info(traceId string, args ...interface{}) {
	l.p(traceId, INFO, args...)
}

func Infof(traceId string, format string, args ...interface{}) {
	l.pf(traceId, INFO, format, args...)
}

func Warning(traceId string, args ...interface{}) {
	l.p(traceId, WARNING, args...)
}

func Warningf(traceId string, format string, args ...interface{}) {
	l.pf(traceId, WARNING, format, args...)
}

func Error(traceId string, args ...interface{}) {
	l.p(traceId, ERROR, args...)
}

func Errorf(traceId string, format string, args ...interface{}) {
	l.pf(traceId, ERROR, format, args...)
}

func Fatal(traceId string, args ...interface{}) {
	l.p(traceId, FATAL, args...)
}

func Fatalf(traceId string, format string, args ...interface{}) {
	l.pf(traceId, FATAL, format, args...)
}

func ForceFlush() {
	l.forceFlush()
}

/* vim: set tabstop=4 set shiftwidth=4 */
