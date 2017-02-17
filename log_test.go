package log

import (
	"errors"
	"testing"
)

func TestLog(t *testing.T) {
	log := Logger("test")
	log.Warning("Danger")
	log.Info("This is important.")
	log.Error(errors.New("this is an error msg"))
	log.DebugTracef("stack trace test: %s", "disabled")
	log.Fatal("FATAL ERROR.  BEEP.")
	log.Info("This is also important.")
	GetGlobalLoggingContext().EnableDebugging(true)
	GetGlobalLoggingContext().SetTracesByDefault(true)
	log.DebugTracef("stack trace test: %s", "enabled")
}