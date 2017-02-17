# log

### Stream-based logging with modular formatters and stack-trace functionality.

```go


func TestLog(t *testing.T) {
	log := Logger("test")
	log.Warning("Danger")
	log.Info("This is important.")
	log.Error(errors.New("this is an error msg"))
	log.DebugTracef("stack trace test: %s", "disabled")
	log.Fatal("FATAL ERROR.  BEEP.")
	log.Info("This is also important.")
	for _, ll := range GetGlobalLoggingContext().GlobalListeners() {
		GetGlobalLoggingContext().AddGlobalLogListener(ll, Trace)
	}
	GetGlobalLoggingContext().EnableDebugging(true)
	GetGlobalLoggingContext().SetTracesByDefault(true)
	log.DebugTracef("stack trace test: %s", "enabled")
}

```

```

 02/17/17 16:13:18.536 | test | Warning | Danger
 02/17/17 16:13:18.536 | test | Info | This is important.
 02/17/17 16:13:18.536 | test | Error | this is an error msg
   this is an error msg
 02/17/17 16:13:18.536 | test | FatalError | FATAL ERROR.  BEEP.
 02/17/17 16:13:18.536 | test | Info | This is also important.
 02/17/17 16:13:18.536 | test | Debug | stack trace test: enabled | /home/dtrombley/go/path/src/github.com/dtromb/log/log_test.go:18
   [0] /home/dtrombley/go/path/src/github.com/dtromb/log/log_test.go:18 in ()
   [1] /home/dtrombley/go/root/src/testing/testing.go:610 in ()
   [2] /home/dtrombley/go/root/src/runtime/asm_amd64.s:2086 in ()

```