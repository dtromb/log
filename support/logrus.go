// +build logrus

package support

// This integration works by inserting /log/ in-between the dispatch and formatting
// components of /logrus/.  That is to say:
//
// A logrus hook is created when a new listener is added via /log/.
//    This hook proxies logrus entries to the /log/ listener, and /log/
//    performs dispatch.  Of course, any /logrus/ configured loggers will 
//    still operate normally.  
//
//    The hook will add stack traces to the JSON object in the logrus log entry
//    if that operation is configured in /log/.
//
//    The hook will add an error containing the logrus-provided JSON object to
//    the /log/ log entry, if the /logrus/ log entry is an error.
//
//
// A proxied logrus formatter is (optionally) inserted as a listener on the 
// new stream via /log/.  Configuration of this formater occurs via the usual
// /logrus/ mechanisms, as well as some provided features (for example, we 
// may translate JSON objects in errors or in the logrus-proxy specific log 
// methods into actual /logrus/ fields.)

import (
	"fmt"
	"time"
	"github.com/dtromb/log"
	"github.com/Sirupsen/logrus"
)

type LogrusLoggingContext struct {
	lock chan bool
	streams map[string]*LogrusLogger
	defaultLogLevel log.LogLevel
	defaultListenerLevel log.LogLevel
	listeners map[log.LogListener]*logrusHook
	debugging bool
	traces bool
	streamsByLogger map[*logrus.Logger]*LogrusLogger
	defaultLogrusStream *LogrusLogger
}

type LogrusLogger struct {
	*logrus.Logger
	ctx *LogrusLoggingContext
	name string
	defaultLogLevel log.LogLevel
	defaultListenerLevel log.LogLevel
	traces bool
	active bool
	listeners map[log.LogListener]*logrusHook
}

func CreateLogrusLoggingContext() *LogrusLoggingContext {
	llc := &LogrusLoggingContext{
		lock: make(chan bool, 1),
		streams: make(map[string]*LogrusLogger),
		defaultLogLevel: log.Info,
		defaultListenerLevel: log.Trace,
		listeners: make(map[log.LogListener]*logrusHook),
		streamsByLogger: make(map[*logrus.Logger]*LogrusLogger),
	}
	llc.lock <- true
	return llc
}

func (ctx *LogrusLoggingContext) getDefaultLogrusStream() *LogrusLogger {
	if ctx.defaultLogrusStream == nil {
		stream := &LogrusLogger{
			Logger: logrus.New(),
			ctx: ctx,
			active: true,
			listeners: make(map[log.LogListener]*logrusHook),
		}
		ctx.defaultLogrusStream = stream
	}
	return ctx.defaultLogrusStream
}
	
func (ctx *LogrusLoggingContext) HasStream(key string) bool {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	if _, has := ctx.streams[key]; has {
		return true
	}
	return false
}

func (ctx *LogrusLoggingContext) Stream(key string) (log.LogStream, bool) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	if stream, has := ctx.streams[key]; has {
		return stream, false
	}
	stream := &LogrusLogger{
		Logger: logrus.New(),
		name: key,
		ctx: ctx,
		active: true,
		listeners: make(map[log.LogListener]*logrusHook),
		defaultLogLevel: log.Default,
		defaultListenerLevel: log.Default,
	}
	ctx.streams[key] = stream
	ctx.streamsByLogger[stream.Logger] = stream
	stream.Logger.Level = logLevelToLogrusLevel(ctx.defaultListenerLevel)
	return stream, true
}

func  (ctx *LogrusLoggingContext) DefaultLogLevel() log.LogLevel {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	return ctx.defaultLogLevel
}

func  (ctx *LogrusLoggingContext) SetDefaultLogLevel(level log.LogLevel) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.defaultLogLevel = level
}

func  (ctx *LogrusLoggingContext) DefaultLogListenerLevel() log.LogLevel {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	return ctx.defaultListenerLevel
}

func  (ctx *LogrusLoggingContext) SetDefaultLogListenerLevel(level log.LogLevel) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.defaultListenerLevel = level
}

func logrusLevelToLogLevel(ll logrus.Level) log.LogLevel {
	switch(ll) {
		case logrus.DebugLevel: return log.Debug
		case logrus.ErrorLevel: return log.Error
		case logrus.FatalLevel: return log.FatalError
		case logrus.InfoLevel: return log.Info
		case logrus.PanicLevel: return log.FatalError
		case logrus.WarnLevel: return log.Warning
	}
	panic("invalid logrus log level")
}

func logLevelToLogrusLevel(ll log.LogLevel) logrus.Level {
	if ll.IsDebug() { return logrus.DebugLevel }
	if ll.IsError() { return logrus.ErrorLevel }
	if ll.IsFatal() { return logrus.FatalLevel }
	if ll.IsInfo() { return logrus.InfoLevel }
	if ll.IsTrace() { return logrus.DebugLevel }
	if ll.IsWarning() { return logrus.WarnLevel }
	panic("invalid log level")
}

type logrusHook struct {
	levels []log.LogLevel
	target log.LogListener
	stream *LogrusLogger
	ctx *LogrusLoggingContext
}

type importLogEntry struct {
	level log.LogLevel
	time time.Time
	stream *LogrusLogger
	message string
	err error
	trace []*log.StackTraceEntry
}

func (lh *logrusHook) Fire(entry *logrus.Entry) error {
	// If the stream is nil, this is a global listener - attempt to 
	// map the logrus logger to a stream, if there is one associated.
	// Otherwise, use the "logrus" default stream.
	var stream log.LogStream
	if lh.stream == nil {
		<-lh.ctx.lock
		defer func() { lh.ctx.lock <- true }()
		if st, has := lh.ctx.streamsByLogger[entry.Logger]; has {
			stream = st
		} else {
			stream = lh.ctx.getDefaultLogrusStream()
		}
	} else {
		stream = lh.stream
	}
	logEntry := &importLogEntry{
		level: logrusLevelToLogLevel(entry.Level),
		time: entry.Time,
		stream: stream.(*LogrusLogger),
		message: entry.Message,
	}
	// XXX - If this is an error, make a LogrusError out of the 
	// fields and associate it here.
	// XXX - Fill in the stack trace here if that is configured.
	lh.target.Receive(logEntry)
	return nil
}	

func (lh *logrusHook) Levels() []logrus.Level {
	levelMap := make(map[logrus.Level]bool)
	for _, l := range lh.levels {
		levelMap[logLevelToLogrusLevel(l)] = true
	}
	res := make([]logrus.Level, 0, len(levelMap)) 
	for l, _ := range levelMap {
		res = append(res, l)
	}
	return res
}

func makeLevelsSlice(minLevel log.LogLevel) []log.LogLevel {
	res := make([]log.LogLevel, 0, int(log.None))
	for i := minLevel; i < log.None; i++ {
		res = append(res, i)
	}
	return res
}

func  (ctx *LogrusLoggingContext) AddGlobalLogListener(logListener log.LogListener, level log.LogLevel) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	listenerHook := &logrusHook{
		target: logListener,
		ctx: ctx,
		levels: makeLevelsSlice(level),
	}
	delete(ctx.listeners,logListener)
	ctx.listeners[logListener] = listenerHook
	// Since this is a global listener, we must update every log stream
	// that is currently available.
	for _, ll := range ctx.streams { 
		// First delete this listener from the hooks.
		for _, level := range logrus.AllLevels {
			for i, h := range ll.Hooks[level] {
				if lh, ok := h.(*logrusHook); ok {
					if lh.target == logListener {
						ll.Hooks[level] = append(ll.Hooks[level][0:i], ll.Hooks[level][i+1:]...)
						break
					}
				}
			}
		}
		// Now re-add this listener with the correct/new set of levels.
		for _, lhl := range listenerHook.levels {
			lrl := logLevelToLogrusLevel(lhl)
			ll.Hooks[lrl] = append(ll.Hooks[lrl], listenerHook)
		}
	}
	// We are done, the logrus -> log global listener proxy is installed.
}

func  (ctx *LogrusLoggingContext) RemoveGlobalLogListener(logListener log.LogListener) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()		
	for _, ll := range ctx.streams { 
		for _, level := range logrus.AllLevels {
			for i, h := range ll.Hooks[level] {
				if lh, ok := h.(*logrusHook); ok {
					if lh.target == logListener {
						ll.Hooks[level] = append(ll.Hooks[level][0:i], ll.Hooks[level][i+1:]...)
						break
					}
				}
			}
		}
	}
	if lh, ok := ctx.listeners[logListener]; ok {
		delete(ctx.streamsByLogger, lh.stream.Logger)
	}
	delete(ctx.listeners, logListener)
}

func  (ctx *LogrusLoggingContext) TracesByDefault() bool {	
	<-ctx.lock
	defer func() { ctx.lock <- true }()		
	return ctx.traces
}

func  (ctx *LogrusLoggingContext) SetTracesByDefault(traces bool) {	
	<-ctx.lock
	defer func() { ctx.lock <- true }()		
	ctx.traces = traces
}

func  (ctx *LogrusLoggingContext) GlobalListeners() []log.LogListener {
	<-ctx.lock
	defer func() { ctx.lock <- true }()		
	res := make([]log.LogListener, len(ctx.listeners))
	for ll, _ := range ctx.listeners {
		res = append(res, ll)
	}
	return res
}

func  (ctx *LogrusLoggingContext) DebuggingEnabled() bool {
	<-ctx.lock
	defer func() { ctx.lock <- true }()		
	return ctx.debugging
}

func  (ctx *LogrusLoggingContext) EnableDebugging(val bool) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()		
	ctx.debugging = val
}

func (ll *LogrusLogger) Logrus() *logrus.Logger {
	return ll.Logger
}

func (ll *LogrusLogger) Log(level log.LogLevel, msg string) {
	lrl := logLevelToLogrusLevel(level)
	if level == log.Default {
		if ll.DefaultLogLevel() == log.Default {
			level = ll.Context().DefaultLogLevel()
		} else {
			level = ll.DefaultLogLevel()
		}
	}
	switch(lrl) {
		case logrus.DebugLevel: ll.Logger.Debug(msg)
		case logrus.ErrorLevel: ll.Logger.Error(msg)
		case logrus.FatalLevel: ll.Logger.Fatal(msg)
		case logrus.InfoLevel: ll.Logger.Info(msg)
		case logrus.WarnLevel: ll.Logger.Warn(msg)
	}
}

func (ll *LogrusLogger) Logf(level log.LogLevel, format string, args ...interface{}) {
	lrl := logLevelToLogrusLevel(level)
	if level == log.Default {
		if ll.DefaultLogLevel() == log.Default {
			level = ll.Context().DefaultLogLevel()
		} else {
			level = ll.DefaultLogLevel()
		}
	}
	switch(lrl) {
		case logrus.DebugLevel: ll.Logger.Debugf(format, args...)
		case logrus.ErrorLevel: ll.Logger.Errorf(format, args...)
		case logrus.FatalLevel: ll.Logger.Fatalf(format, args...)
		case logrus.InfoLevel: ll.Logger.Infof(format, args...)
		case logrus.WarnLevel: ll.Logger.Warnf(format, args...)
	}
}
type StackTraceEntryPresentation struct {
	Pc string			`json:"Pc"`
	Filename string		`json:"Filename"`
	Line int			`json:"Line"`
	FunctionName string	`json:"FunctionName,omitempty"`
}

func stackTraceEntryToJsonPresentation(ste *log.StackTraceEntry) *StackTraceEntryPresentation {
	return &StackTraceEntryPresentation{
		Pc: fmt.Sprintf("0x%16.16X", uint64(ste.Pc())),
		Filename: ste.File(),
		Line: ste.Line(),
		FunctionName: ste.Function().Name(),
	}
}

func (ll *LogrusLogger) LogTracef(level log.LogLevel, format string, args ...interface{}) {
	trace := log.GenerateStackTrace()
	stack := make([]StackTraceEntryPresentation, len(trace)) 
	for i, t := range trace {
		stack[i] = *stackTraceEntryToJsonPresentation(t)
	}
	e := ll.Logger.WithField("_trace", stack)
	lrl := logLevelToLogrusLevel(level)
	if level == log.Default {
		if ll.DefaultLogLevel() == log.Default {
			level = ll.Context().DefaultLogLevel()
		} else {
			level = ll.DefaultLogLevel()
		}
	}
	switch(lrl) {
		case logrus.DebugLevel: e.Debugf(format, args...)
		case logrus.ErrorLevel: e.Errorf(format, args...)
		case logrus.FatalLevel: e.Fatalf(format, args...)
		case logrus.InfoLevel: e.Infof(format, args...)
		case logrus.WarnLevel: e.Warnf(format, args...)
	}	
}

func (ll *LogrusLogger) LogTrace(level log.LogLevel, format string) {
	ll.LogTracef(level, format)
}

func (ll *LogrusLogger) Fatal(msg string) {
	ll.Log(log.FatalError, msg)
}

func (ll *LogrusLogger) Fatalf(format string, args ...interface{}) {
	ll.Logf(log.FatalError, format, args...)
}

func (ll *LogrusLogger) FatalTrace(msg string) {
	ll.LogTrace(log.FatalError, msg)
}

func (ll *LogrusLogger) FatalTracef(format string, args ...interface{}) {
	ll.LogTracef(log.FatalError, format, args...)
}

func (ll *LogrusLogger) Error(err error) {
	ll.WithError(err).Error()
}

func (ll *LogrusLogger) Errorf(err error, format string, args ...interface{}) {
	ll.WithError(err).Errorf(format, args...)
}

func (ll *LogrusLogger) Warning(msg string) {
	ll.Log(log.Warning, msg)
}

func (ll *LogrusLogger) Warningf(format string, args ...interface{}) {
	ll.Logf(log.Warning, format, args...)
}

func (ll *LogrusLogger) WarningTrace(msg string) {
	ll.LogTrace(log.Warning, msg)
}

func (ll *LogrusLogger) WarningTracef(format string, args ...interface{}) {
	ll.LogTracef(log.Warning, format, args...)
}

func (ll *LogrusLogger) Info(msg string) {
	ll.Log(log.Info, msg)
}

func (ll *LogrusLogger) Infof(format string, args ...interface{}) {
	ll.Logf(log.Info, format, args...)
}

func (ll *LogrusLogger) InfoTrace(msg string) {
	ll.LogTrace(log.Info, msg)
}

func (ll *LogrusLogger) InfoTracef(format string, args ...interface{}) {
	ll.LogTracef(log.Info, format, args...)
}

func (ll *LogrusLogger) Debug(msg string) {
	ll.Log(log.Debug, msg)
}

func (ll *LogrusLogger) Debugf(format string, args ...interface{}) {
	ll.Logf(log.Debug, format, args)
}

func (ll *LogrusLogger) DebugTrace(msg string) {
	ll.LogTrace(log.Debug, msg)
}

func (ll *LogrusLogger) DebugTracef(format string, args ...interface{}) {
	ll.LogTracef(log.Debug, format, args...)
}

func (ll *LogrusLogger) Trace(msg string) {
	ll.LogTrace(log.Trace, msg)
}

func (ll *LogrusLogger) Tracef(format string, args ...interface{}) {
	ll.LogTracef(log.Trace, format, args...)
}

func (ll *LogrusLogger) Context() log.LoggingContext {
	return ll.ctx
}

func (ll *LogrusLogger) Name() string {
	return ll.name
}

func (ll *LogrusLogger) DefaultLogLevel() log.LogLevel {
	return ll.defaultLogLevel
}

func (ll *LogrusLogger) SetDefaultLogLevel(level log.LogLevel) {
	ll.defaultLogLevel = level
}

func (ll *LogrusLogger) DefaultLogListenerLevel() log.LogLevel {
	return ll.defaultListenerLevel
}

func (ll *LogrusLogger) SetDefaultLogListenerLevel(level log.LogLevel) {
	ll.defaultListenerLevel = level
}

func (ll *LogrusLogger) AddLogListener(logListener log.LogListener, level log.LogLevel) {
	listenerHook := &logrusHook{
		target: logListener,
		ctx: ll.ctx,
		levels: makeLevelsSlice(level),
		stream: ll,
	}
	delete(ll.listeners,logListener)
	ll.listeners[logListener] = listenerHook
	// First delete this listener from the hooks.
	for _, level := range logrus.AllLevels {
		for i, h := range ll.Hooks[level] {
			if lh, ok := h.(*logrusHook); ok {
				if lh.target == logListener {
					ll.Hooks[level] = append(ll.Hooks[level][0:i], ll.Hooks[level][i+1:]...)
					break
				}
			}
		}
		}
	// Now re-add this listener with the correct/new set of levels.
	for _, lhl := range listenerHook.levels {
		lrl := logLevelToLogrusLevel(lhl)
		ll.Hooks[lrl] = append(ll.Hooks[lrl], listenerHook)
	}
	// We are done, the logrus -> log listener proxy is installed.
}

func (ll *LogrusLogger) RemoveLogListener(logListener log.LogListener) {
	for _, level := range logrus.AllLevels {
		for i, h := range ll.Hooks[level] {
			if lh, ok := h.(*logrusHook); ok {
				if lh.target == logListener {
					ll.Hooks[level] = append(ll.Hooks[level][0:i], ll.Hooks[level][i+1:]...)
					break
				}
			}
		}
	}
	delete(ll.listeners, logListener)
}

func (ll *LogrusLogger) TracesByDefault() bool {
	return ll.traces
}

func (ll *LogrusLogger) SetTracesByDefault(traces bool) {
	ll.traces = traces
}

func (ll *LogrusLogger) IsActive() bool {
	return ll.active
}

func (ll *LogrusLogger) Shutdown() {
	// XXX - implement
}

func (le *importLogEntry) LogTime() time.Time {
	return le.time
}

func (le *importLogEntry) Stream() string {
	return le.stream.name
}

func (le *importLogEntry) Level() log.LogLevel {
	return le.level
}

func (le *importLogEntry) Message() string {
	return le.message
}

func (le *importLogEntry) HasAssociatedError() bool {
	return le.err != nil
}

func (le *importLogEntry) AssociatedError() error {
	return le.err
}

func (le *importLogEntry) HasTrace() bool {
	return le.trace != nil
}

func (le *importLogEntry) Trace() []*log.StackTraceEntry {
	return le.trace
}