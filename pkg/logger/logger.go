package logger

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Debugging
type logTopic string

const (
	DClient logTopic = "CLNT"
	DCommit logTopic = "CMIT"
	DDrop   logTopic = "DROP"
	DError  logTopic = "ERRO"
	DInfo   logTopic = "INFO"
	DWarn   logTopic = "WARN"
	DLog    logTopic = "LOG1"
	DLog2   logTopic = "LOG2"
	DTest   logTopic = "TEST"
	DTimer  logTopic = "TIMR"
	DTrace  logTopic = "TRCE"
)

var debugStart time.Time
var debugVerbosity int

// Retrieve the verbosity level from an environment variable
func getVerbosity() int {
	v := os.Getenv("VERBOSE")
	level := 0
	if v != "" {
		var err error
		level, err = strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Invalid verbosity %v", v)
		}
	}
	return level
}

func DInit() {
	debugStart = time.Now()
	debugVerbosity = getVerbosity()
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func DPrintf(topic logTopic, format string, a ...interface{}) {
	if debugVerbosity > 0 {
		time := time.Since(debugStart).Microseconds()
		time /= 100
		prefix := fmt.Sprintf("%06d %v ", time, string(topic))
		format = prefix + format
		log.Printf(format, a...)
	}
}
