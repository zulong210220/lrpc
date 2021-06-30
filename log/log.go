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

func Debug(args ...interface{}) {
	l.p(DEBUG, args...)
}

func Debugf(format string, args ...interface{}) {
	l.pf(DEBUG, format, args...)
}

func Info(args ...interface{}) {
	l.p(INFO, args...)
}

func Infof(format string, args ...interface{}) {
	l.pf(INFO, format, args...)
}

func Warning(args ...interface{}) {
	l.p(WARNING, args...)
}

func Warningf(format string, args ...interface{}) {
	l.pf(WARNING, format, args...)
}

func Error(args ...interface{}) {
	l.p(ERROR, args...)
}

func Errorf(format string, args ...interface{}) {
	l.pf(ERROR, format, args...)
}

func Fatal(args ...interface{}) {
	l.p(FATAL, args...)
}

func Fatalf(format string, args ...interface{}) {
	l.pf(FATAL, format, args...)
}

func ForceFlush() {
	l.forceFlush()
}

/* vim: set tabstop=4 set shiftwidth=4 */
