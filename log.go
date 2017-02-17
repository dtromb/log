package log

import (
	"fmt"
	"time"
)

type LogEntry interface {
	LogTime() time.Time
	Stream() string
	Level() LogLevel
	Message() string
	HasAssociatedError() bool
	AssociatedError() error
	HasTrace() bool
	Trace() []*StackTraceEntry
}

type LogEntryFormatter interface {
	Format(entry LogEntry) string
}

type LoggingContext interface {
	HasStream(key string) bool
	Stream(key string) (LogStream, bool)
	DefaultLogLevel() LogLevel
	SetDefaultLogLevel(level LogLevel)
	DefaultLogListenerLevel() LogLevel
	SetDefaultLogListenerLevel(level LogLevel)
	AddGlobalLogListener(logListener LogListener, level LogLevel)
	RemoveGlobalLogListener(logListener LogListener)
	TracesByDefault() bool
	SetTracesByDefault(traces bool)
	GlobalListeners() []LogListener
	DebuggingEnabled() bool
	EnableDebugging(val bool)
}
type Log interface {
	Log(level LogLevel, msg string)
	Logf(level LogLevel, format string, args ...interface{})
	LogTrace(level LogLevel, msg string)
	LogTracef(level LogLevel, format string, args ...interface{})
	Fatal(msg string)
	Fatalf(format string, args ...interface{})
	FatalTrace(msg string)
	FatalTracef(format string, args ...interface{})
	Error(err error)
	Errorf(err error, format string, args ...interface{})
	Warning(msg string)
	Warningf(format string, args ...interface{})
	WarningTrace(msg string)
	WarningTracef(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	InfoTrace(msg string)
	InfoTracef(format string, args ...interface{})
	Debug(msg string) 
	Debugf(format string, args ...interface{})
	DebugTrace(msg string) 
	DebugTracef(format string, args ...interface{})
	Trace(msg string)
	Tracef(format string, args ...interface{})
}

type LogStream interface {
	Log
	Context() LoggingContext
	Name() string
	DefaultLogLevel() LogLevel
	SetDefaultLogLevel(level LogLevel)
	DefaultLogListenerLevel() LogLevel
	SetDefaultLogListenerLevel(level LogLevel)
	AddLogListener(logListener LogListener, level LogLevel)
	RemoveLogListener(logListener LogListener)
	TracesByDefault() bool
	SetTracesByDefault(traces bool)
	IsActive() bool
	Shutdown()
}

type LogLevel uint8 
const (
	All			LogLevel = iota
	FatalError
	Error
	Error2 
	Error3
	Warning
	Warning2
	Warning3
	Info
	Info2
	Info3
	Debug
	Debug2
	Debug3
	Debug4
	Debug5
	Trace
	None
	Default
)
func (ll LogLevel) String() string {
	switch(ll) {
		case	 All	: return "All"
		case	 FatalError: return "FatalError"
		case	 Error: return "Error"
		case	 Error2: return "Error-2"
		case	 Error3: return "Error-3"
		case	 Warning: return "Warning"
		case	 Warning2: return "Warning-2"
		case	 Warning3: return "Warning-3"
		case	 Info: return "Info"
		case	 Info2: return "Info-2"
		case	 Info3: return "Info-3"
		case	 Debug: return "Debug"
		case	 Debug2: return "Debug-2"
		case	 Debug3: return "Debug-3"
		case	 Debug4: return "Debug-4"
		case	 Debug5: return "Debug-5"
		case	 Trace: return "Trace"
		case	 None: return "None"
	}
	panic("invalid log level")
}

func (ll LogLevel) IsFatal() bool {
	return ll == FatalError
}

func (ll LogLevel) IsError() bool {
	switch(ll) {
		case Error: return true
		case Error2: return true
		case Error3: return true
	}
	return false
}

func (ll LogLevel) IsWarning() bool {
	switch(ll) {
		case Warning: return true
		case Warning2: return true
		case Warning3: return true
	}
	return false
}


func (ll LogLevel) IsInfo() bool {
	switch(ll) {
		case Info: return true
		case Info2: return true
		case Info3: return true
	}
	return false
}

func (ll LogLevel) IsDebug() bool {
	switch(ll) {
		case Debug: return true
		case Debug2: return true
		case Debug3: return true
		case Debug4: return true
		case Debug5: return true
	}
	return false
}

func (ll LogLevel) IsTrace() bool {
	return ll == Trace
}


///

type stdLoggingContext struct {
	lock chan bool
	debugging bool
	streams map[string]*stdLogStream
	defaultLogLevel LogLevel
	defaultListenerLevel LogLevel
	listeners map[LogListener]LogLevel
	traces bool
}

type stdLogStream struct {
	lock chan bool
	ctx *stdLoggingContext
	name string
	defaultLevel LogLevel
	defaultListenerLevel LogLevel
	listeners map[LogListener]LogLevel
	traces bool
	active bool
}

type stdLogEntry struct {
	ts time.Time
	stream LogStream
	level LogLevel
	message string
	associatedError error
	stackTrace []*StackTraceEntry	
}

func CreateLoggingContext() LoggingContext {
	ctx := &stdLoggingContext{
		lock: make(chan bool, 1),
		streams: make(map[string]*stdLogStream),
		defaultLogLevel: Info,
		listeners: make(map[LogListener]LogLevel),
	}
	ctx.lock <- true
	return ctx
}

func (ctx *stdLoggingContext) HasStream(key string) bool {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	_, has := ctx.streams[key]
	return has
}

func (ctx *stdLoggingContext) Stream(key string) (LogStream, bool) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	stream, has := ctx.streams[key]
	if has {
		return stream, false
	}
	// We will create a new log stream.
	ns := &stdLogStream{
		lock: make(chan bool, 1),
		ctx: ctx,
		name: key,
		defaultLevel: Default,
		defaultListenerLevel: Default,
		listeners: make(map[LogListener]LogLevel),
		traces: false,
		active: true,
	}
	ns.lock <- true
	return ns, true
}

func (ctx *stdLoggingContext) GlobalListeners() []LogListener {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	res := make([]LogListener, 0, len(ctx.listeners))
	for ll, _ := range(ctx.listeners) {
		res = append(res, ll)	
	}
	return res
}

func (ctx *stdLoggingContext) DebuggingEnabled() bool {	
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	return ctx.debugging
}

func (ctx *stdLoggingContext) EnableDebugging(val bool) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	ctx.debugging = val
}

func (ctx *stdLoggingContext) DefaultLogLevel() LogLevel {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	return ctx.defaultLogLevel
}

func (ctx *stdLoggingContext) SetDefaultLogLevel(level LogLevel) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	ctx.defaultLogLevel = level
}

func (ctx *stdLoggingContext) DefaultLogListenerLevel() LogLevel {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	return ctx.defaultListenerLevel
}

func (ctx *stdLoggingContext) SetDefaultLogListenerLevel(level LogLevel) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	ctx.defaultListenerLevel = level
}

func (ctx *stdLoggingContext) AddGlobalLogListener(logListener LogListener, level LogLevel) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	delete(ctx.listeners, logListener)
	ctx.listeners[logListener] = level
}

func (ctx *stdLoggingContext) RemoveGlobalLogListener(logListener LogListener) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	delete(ctx.listeners, logListener)
}


func (ctx *stdLoggingContext) TracesByDefault() bool {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	return ctx.traces
}

func (ctx *stdLoggingContext) SetTracesByDefault(traces bool) {
	<-ctx.lock 
	defer func() { ctx.lock <- true }()
	ctx.traces = traces
}

func (ls *stdLogStream) Context() LoggingContext {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	return ls.ctx
}

func (ls *stdLogStream) DefaultLogLevel() LogLevel {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	return ls.defaultLevel
}

func (ls *stdLogStream) SetDefaultLogLevel(level LogLevel) {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	ls.defaultLevel = level
}

func (ls *stdLogStream) DefaultLogListenerLevel() LogLevel {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	return ls.defaultListenerLevel
}

func (ls *stdLogStream) SetDefaultLogListenerLevel(level LogLevel) {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	ls.defaultListenerLevel = level
}

func (ls *stdLogStream) AddLogListener(logListener LogListener, level LogLevel) {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	delete(ls.listeners, logListener)
	ls.listeners[logListener] = level
}

func (ls *stdLogStream) RemoveLogListener(logListener LogListener) {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	delete(ls.listeners, logListener)
}

func (ls *stdLogStream) TracesByDefault() bool {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	return ls.traces
}

func (ls *stdLogStream) SetTracesByDefault(traces bool) {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	ls.traces = traces
}

func (ls *stdLogStream) IsActive() bool {
	<-ls.lock 
	defer func() { ls.lock <- true }()
	return ls.active
}

func (ls *stdLogStream) Name() string {
	return ls.name
}

func (ls *stdLogStream) Shutdown() {
	panic("stdLogStream.Shutdown() unimplemented")
}

func lockChan(c chan bool) { <- c }
func unlockChan(c chan bool) {
	select {
		case c <- true:
		default: 
	}
}

func (ls *stdLogStream) Log(level LogLevel, msg string) {
	ls.dispatchLog(level, false, nil, msg)
}
func (ls *stdLogStream) Logf(level LogLevel, format string, args ...interface{}) {
	ls.dispatchLog(level, false, nil, format, args...)
}

func (ls *stdLogStream) dispatchLog(level LogLevel, generateTrace bool, setError error, format string, args ...interface{}) {
	ts := time.Now()
	// First assess interest - no point in doing the formatting
	// if no loggers will receive.
	lockChan(ls.lock)
	defer unlockChan(ls.lock)
	lockChan(ls.ctx.lock)
	defer unlockChan(ls.ctx.lock)
	interest := make([]LogListener, 0, 8)
	for ll, lv := range ls.listeners {
		if lv >= level || (lv == Default && ls.ctx.defaultListenerLevel <= level) || level == All {
			interest = append(interest, ll)
		}
	}
	for ll, lv := range ls.ctx.listeners {
		//fmt.Printf("lv: %s level: %s show: ", lv.String(), level.String())
		//fmt.Println(lv >= level)
		if lv >= level || (lv == Default && ls.ctx.defaultListenerLevel <= level) || level == All {
			interest = append(interest, ll)
		}
	}
	unlockChan(ls.ctx.lock)
	if len(interest) > 0 {
		var msg string
		if len(args) > 0 {
			msg = fmt.Sprintf(format, args...)
		} else {
			msg = format
		}
		entry := &stdLogEntry{
			ts: ts,
			stream: ls,
			level: level,
			message: msg,
		}
		if ls.traces || ls.ctx.traces || generateTrace {
			entry.stackTrace = generateStackTrace()
		}
		if setError != nil {
			entry.associatedError = setError
		}
		unlockChan(ls.lock)
		for _, ll := range interest {
			// go ll.Receive(entry)
			ll.Receive(entry)
		}
	}
}

func (ls *stdLogStream) LogTrace(level LogLevel, msg string) {
	ls.dispatchLog(level, true, nil, msg)
}

func (ls *stdLogStream) LogTracef(level LogLevel, format string, args ...interface{}) {
	ls.dispatchLog(level, true, nil, format, args...)
}

func (ls *stdLogStream) Fatal(msg string) {
	ls.dispatchLog(FatalError, false, nil, msg)
}

func (ls *stdLogStream) Fatalf(format string, args ...interface{}) {
	ls.dispatchLog(FatalError, false, nil, format, args...)
}

func (ls *stdLogStream) FatalTrace(msg string) {
	ls.dispatchLog(FatalError, true, nil, msg)
}

func (ls *stdLogStream) FatalTracef(format string, args ...interface{}) {
	ls.dispatchLog(FatalError, true, nil, format, args...)
}

func (ls *stdLogStream) Error(err error) {
	ls.dispatchLog(Error, false, err, err.Error())
}
func (ls *stdLogStream) Errorf(err error, format string, args ...interface{}) {
	ls.dispatchLog(Error, false, err, format, args...)
}

func (ls *stdLogStream) Warning(msg string) {
	ls.dispatchLog(Warning, false, nil, msg)
}

func (ls *stdLogStream) Warningf(format string, args ...interface{}) {
	ls.dispatchLog(Warning, false, nil, format, args...)
}

func (ls *stdLogStream) WarningTrace(msg string) {
	ls.dispatchLog(Warning, true, nil, msg)
}

func (ls *stdLogStream) WarningTracef(format string, args ...interface{}) {
	ls.dispatchLog(Warning, true, nil, format, args...)
}

func (ls *stdLogStream) Info(msg string) {
	ls.dispatchLog(Info, false, nil, msg)
}

func (ls *stdLogStream) Infof(format string, args ...interface{}) {
	ls.dispatchLog(Info, false, nil, format, args...)
}

func (ls *stdLogStream) InfoTrace(msg string) {
	ls.dispatchLog(Info, true, nil, msg)
}

func (ls *stdLogStream) InfoTracef(format string, args ...interface{}) {
	ls.dispatchLog(Info, true, nil, format, args...)
}

func (ls *stdLogStream) Debug(msg string) {
	if ls.ctx.debugging {
		ls.dispatchLog(Debug, false, nil, msg)
	}
}

func (ls *stdLogStream) Debugf(format string, args ...interface{}) {
	if ls.ctx.debugging {
		ls.dispatchLog(Debug, false, nil, format, args...)
	}
}

func (ls *stdLogStream) DebugTrace(msg string) {
	if ls.ctx.debugging {
		ls.dispatchLog(Debug, true, nil, msg)
	}
}

func (ls *stdLogStream) DebugTracef(format string, args ...interface{}) {
	if ls.ctx.debugging {
		ls.dispatchLog(Debug, true, nil, format, args...)
	}
}

func (ls *stdLogStream) Trace(msg string) {
	if ls.ctx.debugging {
		ls.dispatchLog(Trace, true, nil, msg)
	}
}

func (ls *stdLogStream) Tracef(format string, args ...interface{}) {
	if ls.ctx.debugging {
		ls.dispatchLog(Trace	, true, nil, format, args...)
	}
}

func (le *stdLogEntry) LogTime() time.Time {
	return le.ts
}

func (le *stdLogEntry) Stream() string {
	return le.stream.Name()
}

func (le *stdLogEntry) Message() string {
	return le.message
}

func (le *stdLogEntry) HasAssociatedError() bool {
	return le.associatedError != nil
}

func (le *stdLogEntry) AssociatedError() error {
	return le.associatedError
}

func (le *stdLogEntry) HasTrace() bool {
	if le.associatedError != nil {
	}
	return le.stackTrace != nil
}

func (le *stdLogEntry) Level() LogLevel {
	return le.level
}

func (le *stdLogEntry) Trace() []*StackTraceEntry {
	if le.stackTrace == nil {
		return nil
	}
	res := make([]*StackTraceEntry, len(le.stackTrace))
	copy(res, le.stackTrace)
	return res
}