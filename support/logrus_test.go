// +build logrus

package support

import (
	"reflect"
	"os"
	"errors"
	"testing"
	_ "github.com/Sirupsen/logrus"
	logp "github.com/dtromb/log"
)

func TestLogrus(t *testing.T) {
	logging := CreateLogrusLoggingContext()
	log, _  := logging.Stream("logrus-integrarion-test")
	log.Infof("This is a test! (type=%s)", reflect.TypeOf(log).String())
	log.Errorf(errors.New("This is an error"), "The error goes to a logrus field...")
	logging.EnableDebugging(true)
	log.Trace("Stack traces also get added to the logrus fields.")
	
	log.AddLogListener(logp.NewWriterLogger("test-writer", os.Stdout, logp.NewLogEntryFormatter()), logp.Warning)
	logrusLogger := log.(*LogrusLogger).Logrus()
	logrusLogger.Warn("The other way also works!")
}
