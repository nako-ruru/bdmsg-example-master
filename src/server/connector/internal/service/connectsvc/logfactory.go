package connectsvc

import (
	"github.com/jcelliott/lumber"
	. "server/connector/internal/config"
	"github.com/someonegg/goutil/gologf"
	"github.com/someonegg/golog"
	"fmt"
	"os"
	"strings"
	"runtime/debug"
)

var log *lumber.MultiLogger
var timerLog *lumber.MultiLogger

func init()  {
	golog.RootLogger.AddPredef("app", "connector")

	err := ParseConfig()
	if err != nil {
		fmt.Printf("ParseConfig, error=%s\n@%s\n", err, debug.Stack())
		os.Exit(1)
		return
	}

	infoDirectory := getParentDirectory(Config.Log.InfoFile)
	if _, err := os.Stat(infoDirectory); err != nil {
		err := os.MkdirAll(infoDirectory, 0711)
		if err != nil {
			fmt.Printf("Error creating directory, error=%s\n", err)
			return
		}
	}

	err = gologf.SetOutput(Config.Log.InfoFile)
	if err != nil {
		fmt.Printf("ParseConfig, error=%s\n@%s\n", err, debug.Stack())
		os.Exit(1)
		return
	}

	initLogger()
	initTimeLogger()
}

func initLogger()  {
	log = lumber.NewMultiLogger()

	consoleLog := lumber.NewConsoleLogger(lumber.INFO)
	log.AddLoggers(consoleLog)

	fileLog, err1 := lumber.NewFileLogger(Config.Log.InfoFile, lumber.INFO, lumber.ROTATE, 50000, 9, 100)
	if err1 == nil {
		log.AddLoggers(fileLog)
	} else {
		log.Error("createLogger, err=%s,\r\n%s", err1, debug.Stack())
	}

	errorDirectory := getParentDirectory(Config.Log.ErrorFile)
	if _, err := os.Stat(errorDirectory); err != nil {
		err := os.MkdirAll(errorDirectory, 0711)
		if err != nil {
			fmt.Printf("Error creating directory, error=%s\n", err)
			return
		}
	}
	fileError, err2 := lumber.NewFileLogger(Config.Log.ErrorFile, lumber.WARN, lumber.ROTATE, 50000, 9, 100)
	if err2 == nil {
		log.AddLoggers(fileError)
	} else {
		log.Error("createLogger, err=%s\r\n%s", err2, debug.Stack())
	}
}

func initTimeLogger()  {
	timerLog = lumber.NewMultiLogger()

	infoDirectory := getParentDirectory(Config.Log.TimerFile)
	if _, err := os.Stat(infoDirectory); err != nil {
		err := os.MkdirAll(infoDirectory, 0711)
		if err != nil {
			fmt.Printf("Error creating directory, error=%s\n", err)
			return
		}
	}
	fileLog, err1 := lumber.NewFileLogger(Config.Log.TimerFile, lumber.INFO, lumber.ROTATE, 50000, 9, 100)
	if err1 == nil {
		timerLog.AddLoggers(fileLog)
	} else {
		log.Error("createLogger, err=%s,\r\n%s", err1, debug.Stack())
	}
}

func GetLogger() (l *lumber.MultiLogger) {
	return log
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func getParentDirectory(dir string) string {
	return substr(dir, 0, strings.LastIndex(dir, "/"))
}