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

There is support for integration with the popular [logrus](https://github.com/Sirupsen/logrus) logging framework:  (build with '-tags logrus')

```go

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
```
```
INFO[0000] This is a test! (type=*support.LogrusLogger) 
ERRO[0000] The error goes to a logrus field...           error="This is an error"
DEBU[0000] Stack traces also get added to the logrus fields.  _trace=[{0x0000000000472404 /home/dtrombley/go/path/src/github.com/dtromb/log/support/logrus.go 467 } {0x0000000000473708 /home/dtrombley/go/path/src/github.com/dtromb/log/support/logrus_test.go 20 } {0x0000000000469E21 /home/dtrombley/go/root/src/testing/testing.go 610 } {0x0000000000457CB1 /home/dtrombley/go/root/src/runtime/asm_amd64.s 2086 }]
02/17/17 23:13:56.629 | logrus-integrarion-test | Warning | The other way also works!
 WARN[0000] The other way also works!       
```




