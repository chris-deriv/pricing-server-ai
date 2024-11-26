package logging

import (
	"log"
	"os"
	"strconv"
)

var debugLogging bool

func init() {
	// Check for DEBUG environment variable, default to false
	debugLog := os.Getenv("DEBUG")
	if parsed, err := strconv.ParseBool(debugLog); err == nil {
		debugLogging = parsed
	}
}

func DebugLog(format string, v ...interface{}) {
	if debugLogging {
		log.Printf(format, v...)
	}
} 