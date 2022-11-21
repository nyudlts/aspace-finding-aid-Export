package aspace_xport

import (
	"fmt"
	"log"
	"os"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

var (
	debug   = false
	logfile = "aspace-export"
)

func getLogLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "[DEBUG]"
	case INFO:
		return "[INFO]"
	case WARNING:
		return "[WARNING]"
	case ERROR:
		return "[ERROR]"
	case FATAL:
		return "[FATAL]"
	default:
		panic(fmt.Errorf("log level %v is not supported", level))
	}
}

func CreateLog(formattedTime string, dbug bool) error {
	//create a log file
	logfile = logfile + "-" + formattedTime + ".log"

	f, err := os.Create(logfile)
	if err != nil {
		return err
	}

	defer f.Close()
	log.SetOutput(f)
	PrintAndLog(fmt.Sprintf("logging to %s", logfile), INFO)
	debug = dbug
	return nil
}

// logging and printing functions
func PrintAndLog(msg string, logLevel LogLevel) {
	if logLevel == DEBUG && debug == false {

	} else {
		level := getLogLevelString(logLevel)
		fmt.Printf("%s %s\n", level, msg)
		log.Printf("%s %s", level, msg)
	}
}

func PrintOnly(msg string, logLevel LogLevel) {
	if logLevel == DEBUG && debug == false {

	} else {
		level := getLogLevelString(logLevel)
		fmt.Printf("%s %s\n", level, msg)
	}
}

func LogOnly(msg string, logLevel LogLevel) {
	if logLevel == DEBUG && debug == false {

	} else {
		level := getLogLevelString(logLevel)
		log.Printf("%s %s\n", level, msg)
	}
}
