package connectsvc

import (
	"github.com/jcelliott/lumber"
)

func createLogger() (l *lumber.MultiLogger) {
	mlog := lumber.NewMultiLogger()
	consoleLog := lumber.NewConsoleLogger(lumber.INFO)
	mlog.AddLoggers(consoleLog)
	fileLog, err1 := lumber.NewFileLogger("log/connector.err", lumber.INFO, lumber.ROTATE, 5000, 9, 100)
	if err1 == nil {
		mlog.AddLoggers(fileLog)
	} else {
		mlog.Error("createLogger, err=%s", err1)
	}
	fileError, err2 := lumber.NewFileLogger("log/connector.err", lumber.ERROR, lumber.ROTATE, 5000, 9, 100)
	if err2 == nil {
		mlog.AddLoggers(fileError)
	} else {
		mlog.Error("createLogger, err=%s", err2)
	}
	return mlog
}
