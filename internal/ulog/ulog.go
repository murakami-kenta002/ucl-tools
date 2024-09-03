package ulog

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	Ldate         = log.Ldate
	Ltime         = log.Ltime
	Lmicroseconds = log.Lmicroseconds
	Llongfile     = log.Llongfile
	Lshortfile    = log.Lshortfile
	LUTC          = log.LUTC
	LstdFlags     = Ldate | Ltime
)

var DLog *Logger // for debug log
var ILog *Logger // for info log
var WLog *Logger // for warn log
var ELog *Logger // for err log

type Logger struct {
	logger    *log.Logger
	calldepth int
}

func New(out io.Writer, prefix string, flag int) *Logger {
	l := log.New(out, prefix, flag)
	return &Logger{logger: l, calldepth: 2}
}

func (l *Logger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
	return
}

func (l *Logger) SetPrefix(prefix string) {
	l.logger.SetPrefix(prefix)
	return
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.logger.Output(l.calldepth, fmt.Sprintf(format, v...))
}

func (l *Logger) Print(v ...interface{}) {
	l.logger.Output(l.calldepth, fmt.Sprint(v...))
}

func (l *Logger) Println(v ...interface{}) {
	l.logger.Output(l.calldepth, fmt.Sprint(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Output(l.calldepth, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.logger.Output(l.calldepth, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.logger.Output(l.calldepth, fmt.Sprint(v...))
	os.Exit(1)
}

func SetLogPrefix(appName string) {
	if len(appName) != 0 {
		appName = appName + " "
	}
	DLog.SetPrefix("[" + appName + "dbg]")
	ILog.SetPrefix("[" + appName + "info]")
	WLog.SetPrefix("[" + appName + "warn]")
	ELog.SetPrefix("[" + appName + "err]")
}

func init() {
	DLog = New(ioutil.Discard, "[dbg]", Lmicroseconds|Lshortfile)
	ILog = New(ioutil.Discard, "[info]", Lmicroseconds|Lshortfile)
	WLog = New(os.Stderr, "[warn]", Lmicroseconds|Lshortfile)
	ELog = New(os.Stderr, "[err]", Lmicroseconds|Lshortfile)
}
