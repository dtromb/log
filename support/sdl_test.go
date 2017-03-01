//+build sdl

package support

import (
	"os"
	"testing"
	"github.com/dtromb/log"
)

func TestSdl(t *testing.T) {
	test_SdlInit()
	ctx := CreateSdlLoggingContext()
	formatter := log.NewLogEntryFormatter()
	if log.IsTerminal(os.Stdout) {
		formatter.SetFlags(log.PrintColor)
	}
	stdoutLogger := log.NewWriterLogger("default-stdout", os.Stdout, formatter)
	ctx.AddGlobalLogListener(stdoutLogger, log.All)
	test_SdlLog("Hello, SDL!")
	test_SdlQuit()
}