package log

import (
	"fmt"
	"unsafe"
	"io"
	"syscall"
	"os"
)

var _GLOBAL_loggingContext LoggingContext
var _GLOBAL_loggingContextLock chan bool = make(chan bool, 1)

func init() {
	GetGlobalLoggingContext()
}

func GetGlobalLoggingContext() LoggingContext {
	_GLOBAL_loggingContextLock <- true
	if _GLOBAL_loggingContext == nil {
		_GLOBAL_loggingContext = CreateLoggingContext()
	
		// Set up a default output stream listener.
		formatter := NewLogEntryFormatter()
		if hasTerminal(os.Stdout) {
			formatter.SetFlags(PrintColor)
		}
		stdoutLogger := NewWriterLogger("default-stdout", os.Stdout, formatter)
		_GLOBAL_loggingContext.AddGlobalLogListener(stdoutLogger, Trace)
		fmt.Println("INIT")
	}
	<-_GLOBAL_loggingContextLock 
	return _GLOBAL_loggingContext
}

func Logger(name string) Log {
	stream, _ := GetGlobalLoggingContext().Stream(name)
	return stream
}

func hasTerminal(writer io.Writer) bool {
	var termios syscall.Termios
    switch v := writer.(type) {
    		case *os.File: {
            _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(v.Fd()), 
					syscall.TCGETS, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
    			return err == 0
		}
	}
	return false
}
