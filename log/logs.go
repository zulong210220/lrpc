package log

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zulong210220/lrpc/utils"
)

const (
	DATA_FMT_DATA = "2006-01-02"
	DATA_FMT_LOG  = "2006-01-02.15"
	LOG_FLAG      = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	LOG_MODE      = 0666
)

type LEVEL byte

const (
	ALL LEVEL = iota + 1
	DEBUG
	INFO
	WARNING
	ERROR
	FATAL
)

var levelText = map[LEVEL]string{
	ALL:     "ALL",
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARNING: "WARNING",
	ERROR:   "ERROR",
	FATAL:   "FATAL",
}

func getLevel(level string) LEVEL {
	switch level {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING":
		return WARNING
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return ALL
	}
}

var l *Logger

type Logger struct {
	fileSize int64
	fileNum  int32
	fileName string
	s        int
	debug    bool
	level    LEVEL
	dir      string
	ch       chan *Atom
	bytePool *sync.Pool
	f        *os.File
	w        *bufio.Writer
	mu       sync.Mutex
}

type Atom struct {
	line    int
	file    string
	format  string
	level   LEVEL
	traceId string
	args    []interface{}
}

func newLogger(config *Config) {
	debug := config.Env == "DEBUG"
	l = &Logger{
		dir:      config.Dir,
		fileSize: config.FileSize * 1024 * 1024,
		fileNum:  config.FileNum,
		fileName: path.Join(config.Dir, config.FileName+".log"),
		debug:    debug,
		level:    getLevel(config.Level),
		bytePool: &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
		ch:       make(chan *Atom, 1024),
	}
	if l.debug {
		return
	}

	//var err error
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	fmt.Println("DEBUG", pwd)

	err = os.MkdirAll(l.dir, 0755)
	if err != nil {
		fmt.Println("MkdirAll", err)
	}

	l.f, err = os.OpenFile(l.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	fileInfo, err := os.Stat(l.fileName)
	if err != nil {
		panic(err)
	}
	l.s = int(fileInfo.Size())
	l.w = bufio.NewWriterSize(l.f, 1024*1024)
}

func (l *Logger) start() {
	defer func() {
		err := recover()
		if err != nil {
			l.w.Write(debug.Stack())
			l.w.WriteString(fmt.Sprint(err))
		}
		l.w.Flush()
		os.Exit(1)
	}()
	for {
		a := <-l.ch
		if a == nil {
			if _, err := os.Stat(l.fileName); err != nil && os.IsNotExist(err) {
				l.f.Close()
				l.f, _ = os.OpenFile(l.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				l.w.Reset(l.f)
				l.s = 0
			}
			l.w.Flush()
			continue
		}
		if l.s > int(l.fileSize) {
			l.w.Flush()
			l.f.Close()
			os.Rename(l.fileName, l.logname())
			l.f, _ = os.OpenFile(l.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			l.w.Reset(l.f)
			l.s = 0
			l.rm()
		}

		l.write(a)
	}
}

func (l *Logger) run() {
	if l.debug {
		return
	}
	go l.flush()
	go l.start()
}

func (l *Logger) stop() {
	if l != nil && l.w != nil {
		l.w.Flush()
	}
}

func (l *Logger) write(a *Atom) {
	now := time.Now()
	t := now.Nanosecond() / 1000
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	n, _ := l.w.Write([]byte(strconv.FormatInt(int64(year), 10)))
	l.s += n
	n, _ = l.w.Write([]byte{'-',
		byte(month/10) + 48, byte(month%10) + 48, '-',
		byte(day/10) + 48, byte(day%10) + 48, ' ',
		byte(hour/10) + 48, byte(hour%10) + 48, ':',
		byte(minute/10) + 48, byte(minute%10) + 48, ':',
		byte(second/10) + 48, byte(second%10) + 48, '.',
		byte((t%1000000)/100000) + 48, byte((t%100000)/10000) + 48, byte((t%10000)/1000) + 48, ' ',
	})
	l.s += n

	// traceId
	l.w.Write([]byte{'['})
	n, _ = l.w.WriteString(a.traceId)
	l.w.Write([]byte{']', ' '})
	l.s += n + 3

	n, _ = l.w.WriteString(levelText[a.level])
	l.s += n
	l.w.WriteByte(' ')
	l.s += 1
	n, _ = l.w.WriteString(a.file)
	l.s += n
	n, _ = l.w.Write([]byte{':', byte((a.line%10000)/1000) + 48, byte((a.line%1000)/100) + 48,
		byte((a.line%100)/10) + 48, byte(a.line%10) + 48, ' '})
	l.s += n

	// go id
	l.w.Write([]byte{'['})
	n, _ = l.w.WriteString(utils.GetGoidStr())
	l.w.Write([]byte{']'})
	l.s += n + 2

	if len(a.format) == 0 {
		l.w.WriteByte(' ')
		l.s++
		// 此处fmt buffer nil panic 数据竞争导致
		n, _ = fmt.Fprint(l.w, a.args...)
		l.s += n
	} else {
		l.w.WriteByte(' ')
		n, _ = fmt.Fprintf(l.w, a.format, a.args...)
		l.s += n
	}

	l.w.WriteByte(10)
	l.s++
}

func (l *Logger) format(a *Atom) (int, []byte) {
	w := l.bytePool.Get().(*bytes.Buffer)
	defer func() {
		w.Reset()
		l.bytePool.Put(w)
	}()
	now := time.Now()
	t := now.Nanosecond() / 1000
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	w.Write([]byte{byte(year/10) + 48, byte(year%10) + 48, '-',
		byte(month/10) + 48, byte(month%10) + 48, '-',
		byte(day/10) + 48, byte(day%10) + 48, ' ',
		byte(hour/10) + 48, byte(hour%10) + 48, ':',
		byte(minute/10) + 48, byte(minute%10) + 48, ':',
		byte(second/10) + 48, byte(second%10) + 48, '.',
		byte((t%1000000)/100000) + 48, byte((t%100000)/10000) + 48, byte((t%10000)/1000) + 48, ' ',
	})
	w.WriteString(levelText[a.level])
	w.WriteByte(' ')
	w.WriteString(a.file)
	w.Write([]byte{':', byte((a.line%10000)/1000) + 48, byte((a.line%1000)/100) + 48,
		byte((a.line%100)/10) + 48, byte(a.line%10) + 48, ' '})
	if len(a.format) == 0 {
		w.WriteByte(' ')
		fmt.Fprint(w, a.args...)
	} else {
		w.WriteByte(' ')
		fmt.Fprintf(w, a.format, a.args...)
	}
	w.WriteByte(10)
	len := w.Len()
	data := make([]byte, len)
	copy(data, w.Bytes())
	return len, data
}

func (l *Logger) rm() {
	out, err := exec.Command("ls", l.dir).Output()
	if err != nil {
		return
	}

	files := bytes.Split(out, []byte("\n"))
	totol, idx := len(files)-1, 0
	for i := totol; i >= 0; i-- {
		file := string(files[i])
		if strings.HasPrefix(file, "INFO.log") {
			idx++
			if idx > int(l.fileNum) {
				exec.Command("rm", path.Join(l.dir, file)).Run()
			}
		}
	}
}

func (l *Logger) flush() {
	for range time.NewTicker(time.Second).C {
		l.ch <- nil
	}
}

func (l *Logger) forceFlush() {
	if l != nil {
		l.ch <- nil
	}
}

func (l *Logger) logname() string {
	t := fmt.Sprintf("%s", time.Now())[:19]
	tt := strings.Replace(
		strings.Replace(
			strings.Replace(t, "-", "", -1),
			" ", "", -1),
		":", "", -1)
	return fmt.Sprintf("%s.%s", l.fileName, tt)
}

func (l *Logger) genTime() string {
	return fmt.Sprintf("%s", time.Now())[:26]
}

func (l *Logger) p(traceId string, level LEVEL, args ...interface{}) {
	file, line := l.getFileNameAndLine()
	if l == nil {
		fmt.Printf("%s %s %s:%d ", l.genTime(), levelText[level], file, line)
		fmt.Println(args...)
		return
	}
	if l.debug {
		l.mu.Lock()
		defer l.mu.Unlock()
		fmt.Printf("%s %s %s:%d ", l.genTime(), levelText[level], file, line)
		fmt.Println(args...)
		return
	}
	if level >= l.level {
		l.ch <- &Atom{traceId: traceId, file: file, line: line, level: level, args: args}
	}
}

func (l *Logger) pf(traceId string, level LEVEL, format string, args ...interface{}) {
	file, line := l.getFileNameAndLine()
	if l == nil {
		fmt.Printf("%s %s %s:%d ", l.genTime(), levelText[level], file, line)
		fmt.Printf(format, args...)
		fmt.Println()
		return
	}
	if l.debug {
		l.mu.Lock()
		defer l.mu.Unlock()
		fmt.Printf("%s %s %s:%d ", l.genTime(), levelText[level], file, line)
		fmt.Printf(format, args...)
		fmt.Println()
		return
	}
	if level >= l.level {
		l.ch <- &Atom{traceId: traceId, file: file, line: line, format: format, level: level, args: args}
	}
}

func (l *Logger) getFileNameAndLine() (string, int) {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "???", 1
	}
	dirs := strings.Split(file, "/")
	if len(dirs) >= 2 {
		return dirs[len(dirs)-2] + "/" + dirs[len(dirs)-1], line
	}
	return file, line
}
