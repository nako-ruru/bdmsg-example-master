package connectsvc

import (
	"github.com/jcelliott/lumber"
	"github.com/someonegg/goutil/gologf"
	"github.com/someonegg/golog"
	"fmt"
	"os"
	"strings"
	"github.com/docker/go-units"
	"runtime/debug"
	"io/ioutil"
	"encoding/json"
)

const TIMEFORMAT = "2006-01-02 15:04:05.000"

var log *lumber.MultiLogger
var timerLog *lumber.MultiLogger

type LogConfigT struct {
	Refresh string    `json:"refresh"`
	Loggers []LoggerT `json:"loggers"`
}
type LoggerT struct {
	Name      string     `json:"name"`
	Appenders []appender `json:"appenders"`
}
type appender struct {
	Enable       bool   `json:"enable"`
	AppenderType string `json:"type"`
	Level        string `json:"level"`
	Filename     string `json:"filename"`
	Pattern      string `json:"pattern"`
	Rotate       bool   `json:"rotate"`
	Maxlines     string `json:"maxlines"`
}

var logger = map[string]*lumber.MultiLogger{}

func init()  {
	golog.RootLogger.AddPredef("app", "connector")

	loggerConfig, err := ParseConfig2()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ParseConfig, error=%s\n@%s\n", err, debug.Stack())
		os.Exit(1)
		return
	}
	rootPath, err := initLogger(loggerConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "initLogger, error=%s\n@%s\n", err, debug.Stack())
		os.Exit(1)
		return
	}

	err = gologf.SetOutput(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ParseConfig, error=%s\n@%s\n", err, debug.Stack())
		os.Exit(1)
		return
	}

	log = logger["root"]
	timerLog = logger["timer"]
}

func initLogger(loggerConfig *LogConfigT) (string, error) {
	var rootPath string

	for _, logDesc := range loggerConfig.Loggers {
		log = lumber.NewMultiLogger()
		for _, appenderDesc := range logDesc.Appenders {
			if appenderDesc.Enable {
				switch appenderDesc.AppenderType {
				case "console":
					consoleLog := lumber.NewConsoleLogger(stringToLevel(appenderDesc.Level))
					consoleLog.TimeFormat(TIMEFORMAT)
					log.AddLoggers(consoleLog)
					break
				case "file":
					dir := getParentDirectory(appenderDesc.Filename)
					if _, err := os.Stat(dir); err != nil {
						err := os.MkdirAll(dir, 0711)
						if err != nil {
							return "", err
						}
					}
					size, err := units.FromHumanSize(appenderDesc.Maxlines)
					if err != nil {
						return "", err
					}
					fileLog, err1 := lumber.NewFileLogger(appenderDesc.Filename, stringToLevel(appenderDesc.Level), lumber.ROTATE, int(size), 9, 100)
					if err1 == nil {
						fileLog.TimeFormat(TIMEFORMAT)
						log.AddLoggers(fileLog)

						if logDesc.Name == "root" && appenderDesc.Level == "INFO" {
							rootPath = appenderDesc.Filename
						}
					} else {
						return "", err1
					}
					break
				}
			}
		}
		logger[logDesc.Name] = log
	}

	return rootPath, nil
}

func GetLogger() (l *lumber.MultiLogger) {
	return log
}

func stringToLevel(levelText string) int {
	switch levelText {
	case "TRACE":
		return lumber.TRACE
	case "DEBUG":
		return lumber.DEBUG
	case "INFO":
		return lumber.INFO
	case "WARN":
		return lumber.WARN
	case "ERROR":
		return lumber.ERROR
	case "FATAL":
		return lumber.FATAL
	default:
		return -1
	}
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
func ParseConfig2() (*LogConfigT, error) {
	f, err := os.Open("./log.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	blob, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	loggerConfig := &LogConfigT{}
	err = json.Unmarshal(blob, loggerConfig)
	if err != nil {
		return nil, err
	}

	return loggerConfig, nil
}
