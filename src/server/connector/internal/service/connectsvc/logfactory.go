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

func init()  {
	golog.RootLogger.AddPredef("app", "connector")

	err := ParseConfig()
	if err != nil {
		fmt.Printf("ParseConfig, error=%s\n", err)
		return
	}

	infoDirectory := getParentDirectory(Config.InfoFile)
	if _, err := os.Stat(infoDirectory); err != nil {
		err := os.MkdirAll(infoDirectory, 0711)
		if err != nil {
			fmt.Printf("Error creating directory, error=%s\n", err)
			return
		}
	}

	errorDirectory := getParentDirectory(Config.ErrorFile)
	if _, err := os.Stat(errorDirectory); err != nil {
		err := os.MkdirAll(errorDirectory, 0711)
		if err != nil {
			fmt.Printf("Error creating directory, error=%s\n", err)
			return
		}
	}

	err = gologf.SetOutput(Config.InfoFile)
	if err != nil {
		fmt.Printf("ParseConfig, error=%s\n", err)
		return
	}

	log = lumber.NewMultiLogger()

	consoleLog := lumber.NewConsoleLogger(lumber.INFO)
	log.AddLoggers(consoleLog)

	fileLog, err1 := lumber.NewFileLogger(Config.InfoFile, lumber.INFO, lumber.ROTATE, 50000, 9, 100)
	if err1 == nil {
		log.AddLoggers(fileLog)
	} else {
		log.Error("createLogger, err=%s,\r\n%s", err1, debug.Stack())
	}

	fileError, err2 := lumber.NewFileLogger(Config.ErrorFile, lumber.WARN, lumber.ROTATE, 50000, 9, 100)
	if err2 == nil {
		log.AddLoggers(fileError)
	} else {
		log.Error("createLogger, err=%s\r\n%s", err2, debug.Stack())
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