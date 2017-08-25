package connectsvc

import (
	"github.com/jcelliott/lumber"
	. "server/connector/internal/config"
	"github.com/someonegg/goutil/gologf"
	"github.com/someonegg/golog"
)

var log *lumber.MultiLogger

func init()  {
	golog.RootLogger.AddPredef("app", "connector")

	err := ParseConfig()
	if err != nil {
		log.Error("main$ParseConfig", "error", err)
		return
	}

	err = gologf.SetOutput(Config.Logfile)
	if err != nil {
		log.Error("main$SetOutput, err=%s", err)
		return
	}

	log = lumber.NewMultiLogger()

	consoleLog := lumber.NewConsoleLogger(lumber.INFO)
	log.AddLoggers(consoleLog)

	fileLog, err1 := lumber.NewFileLogger(Config.Logfile, lumber.INFO, lumber.ROTATE, 5000, 9, 100)
	if err1 == nil {
		log.AddLoggers(fileLog)
	} else {
		log.Error("createLogger, err=%s", err1)
	}

	fileError, err2 := lumber.NewFileLogger(Config.LogErrorFile, lumber.ERROR, lumber.ROTATE, 5000, 9, 100)
	if err2 == nil {
		log.AddLoggers(fileError)
	} else {
		log.Error("createLogger, err=%s", err2)
	}
}

func GetLogger() (l *lumber.MultiLogger) {
	return log
}