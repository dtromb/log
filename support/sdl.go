//+build sdl

package support

import (
	"time"
	"runtime"
	"fmt"
	"unsafe"
	"github.com/dtromb/log"
)

/*
	#cgo pkg-config: sdl2	
	#include <SDL.h>
	#include <SDL_log.h>
	
	extern sdlLogOutputDispatch(char *userdata, int category, SDL_LogPriority pri, char *msg);
		
	void cgo_sdl_log_output_dispatch_impl(void *userdata, int category, SDL_LogPriority pri, const char *msg) {
		sdlLogOutputDispatch((char*)userdata, category, pri, (char*)msg);
	}
	SDL_LogOutputFunction cgo_sdl_log_output_dispatch = cgo_sdl_log_output_dispatch_impl;

	void cgo_sdl_log(const char *msg) {
		SDL_Log("%s", msg);
	}
	
	void cgo_sdl_set_log_output_function(SDL_LogOutputFunction f, void *d) {
		SDL_LogSetOutputFunction(f,d);
	}
	
	void cgo_sdl_log_message(int category, SDL_LogPriority priority, const char *msg) {
		SDL_LogMessage(category, priority, "%s", msg);
	}

*/
import "C"

type SdlLoggingContext struct {
	lock chan bool
	customStreams map[string]log.LogStream
	customStreamsByCode map[int]string
	stdStreams map[SdlLogContextName]log.LogStream
	defaultLevel log.LogLevel
	defaultListenerLevel log.LogLevel
	listeners map[log.LogListener]log.LogLevel
	debugEnabled bool
	traces bool
	handleId int
}

type SdlLogStream struct {
	ctx *SdlLoggingContext
	name string
	categoryCode int
	defaultLevel log.LogLevel
	defaultListenerLevel log.LogLevel
	listeners map[log.LogListener]log.LogLevel
	traces bool
}

type sdlLogEntry struct {
	timestamp time.Time
	stream SdlLogContextName
	level log.LogLevel
	msg string
}

type SdlLogUserdata struct {
	lock chan bool
	contexts map[int]*SdlLoggingContext
	nextHandle int
}

var global_SdlLogUserdata *SdlLogUserdata = &SdlLogUserdata{
	lock: make(chan bool, 1),
	contexts: make(map[int]*SdlLoggingContext),
	nextHandle: 1,
}

func init() {
	global_SdlLogUserdata.lock <- true
}

func CreateSdlLoggingContext() *SdlLoggingContext {
	ctx := &SdlLoggingContext{
		lock: make(chan bool, 1),
		customStreams: make(map[string]log.LogStream),
		customStreamsByCode: make(map[int]string),
		stdStreams: make(map[SdlLogContextName]log.LogStream),
		defaultLevel: log.Info,
		defaultListenerLevel: log.Trace,	
		listeners: make(map[log.LogListener]log.LogLevel),
	}
	for _, key := range AllSdlLogContextNames() {
		nls := &SdlLogStream{
			ctx: ctx,
			name: string(key),
			categoryCode: int(key.Code()),
		}
		ctx.stdStreams[key] = nls
	}
	<-global_SdlLogUserdata.lock
	ctx.handleId = global_SdlLogUserdata.nextHandle
	global_SdlLogUserdata.nextHandle++
	global_SdlLogUserdata.contexts[ctx.handleId] = ctx
	defer func() { global_SdlLogUserdata.lock <- true }()
	C.SDL_LogSetOutputFunction(C.cgo_sdl_log_output_dispatch, unsafe.Pointer(uintptr(ctx.handleId)))
	runtime.SetFinalizer(ctx, clearGlobalContext)
	ctx.lock <- true
	return ctx
}

func clearGlobalContext(ctx *SdlLoggingContext) {
	<-global_SdlLogUserdata.lock
	defer func() { global_SdlLogUserdata.lock <- true }()
	delete(global_SdlLogUserdata.contexts, ctx.handleId)
}

type SdlLogContextName string 
const(
	SdlLogContextApplication 	= SdlLogContextName("application")
	SdlLogContextError 			= SdlLogContextName("error")
	SdlLogContextAssert 		= SdlLogContextName("assert")
	SdlLogContextSystem 		= SdlLogContextName("system")
	SdlLogContextAudio 			= SdlLogContextName("audio")
	SdlLogContextVideo 			= SdlLogContextName("video")
	SdlLogContextRender 		= SdlLogContextName("render")
	SdlLogContextInput 			= SdlLogContextName("input")
	SdlLogContextTest 			= SdlLogContextName("test")
)

func SdlStdLogContextByCode(code int) (SdlLogContextName, bool) {
	switch(code) {
		case int(C.SDL_LOG_CATEGORY_APPLICATION): 	return SdlLogContextApplication, true
		case int(C.SDL_LOG_CATEGORY_ERROR): 		return SdlLogContextError, true
		case int(C.SDL_LOG_CATEGORY_ASSERT): 		return SdlLogContextAssert, true
		case int(C.SDL_LOG_CATEGORY_SYSTEM): 		return SdlLogContextSystem, true
		case int(C.SDL_LOG_CATEGORY_AUDIO): 		return SdlLogContextAudio, true
		case int(C.SDL_LOG_CATEGORY_VIDEO): 		return SdlLogContextVideo, true
		case int(C.SDL_LOG_CATEGORY_RENDER): 		return SdlLogContextRender, true
		case int(C.SDL_LOG_CATEGORY_INPUT): 		return SdlLogContextInput, true
		case int(C.SDL_LOG_CATEGORY_TEST): 			return SdlLogContextTest, true
	}
	return SdlLogContextName(""), false
}

func AllSdlLogContextNames() []SdlLogContextName {
	return []SdlLogContextName{
		SdlLogContextApplication, 	
		SdlLogContextError, 			
		SdlLogContextAssert, 		
		SdlLogContextSystem, 		
		SdlLogContextAudio, 			
		SdlLogContextVideo, 			
		SdlLogContextRender,		
		SdlLogContextInput, 		
		SdlLogContextTest, 		
	}
}

func (cn SdlLogContextName) Code() C.int {
	switch(cn) {
		case SdlLogContextApplication:  	return C.SDL_LOG_CATEGORY_APPLICATION
		case SdlLogContextError:  			return C.SDL_LOG_CATEGORY_ERROR
		case SdlLogContextAssert:  			return C.SDL_LOG_CATEGORY_ASSERT
		case SdlLogContextSystem:  			return C.SDL_LOG_CATEGORY_SYSTEM
		case SdlLogContextAudio:  			return C.SDL_LOG_CATEGORY_AUDIO
		case SdlLogContextVideo:  			return C.SDL_LOG_CATEGORY_VIDEO
		case SdlLogContextRender:  			return C.SDL_LOG_CATEGORY_RENDER
		case SdlLogContextInput:  			return C.SDL_LOG_CATEGORY_INPUT
		case SdlLogContextTest:  			return C.SDL_LOG_CATEGORY_TEST
	}
	return C.SDL_LOG_CATEGORY_CUSTOM
}

func (cn SdlLogContextName) Custom() bool {
	return cn.Code() != C.SDL_LOG_CATEGORY_CUSTOM
}

type SdlLogPriority C.SDL_LogPriority
const (
	SdlLogPriorityVerbose		SdlLogPriority = C.SDL_LOG_PRIORITY_VERBOSE
	SdlLogPriorityDebug			SdlLogPriority = C.SDL_LOG_PRIORITY_DEBUG
	SdlLogPriorityInfo			SdlLogPriority = C.SDL_LOG_PRIORITY_INFO
	SdlLogPriorityWarn			SdlLogPriority = C.SDL_LOG_PRIORITY_WARN
	SdlLogPriorityError 		SdlLogPriority = C.SDL_LOG_PRIORITY_ERROR
	SdlLogPriorityCritical		SdlLogPriority = C.SDL_LOG_PRIORITY_CRITICAL
)

func (lp SdlLogPriority) Level() log.LogLevel {
	switch(lp) {
		case SdlLogPriorityVerbose: return log.Trace
		case SdlLogPriorityDebug: return log.Debug
		case SdlLogPriorityInfo: return log.Info
		case SdlLogPriorityWarn: return log.Warning
		case SdlLogPriorityError: return log.Error
		case SdlLogPriorityCritical: return log.FatalError
	}
	return log.None
}

func SdlLogPriorityForLogLevel(level log.LogLevel) SdlLogPriority {
	if level.IsFatal() {
		return SdlLogPriorityCritical
	} else if level.IsWarning() {
		return SdlLogPriorityWarn
	} else if level.IsError() {
		return SdlLogPriorityError
	} else if level.IsInfo() {
		return SdlLogPriorityInfo
	} else if level.IsDebug() {
		return SdlLogPriorityDebug
	} else {
		return SdlLogPriorityVerbose
	}
}

func (ctx *SdlLoggingContext) getCategoryByCode(code int) (SdlLogContextName,bool) {
	cat, isStd := SdlStdLogContextByCode(code)
	if !isStd {
		name, has := ctx.customStreamsByCode[code]
		if !has {
			return "", false
		}
		return SdlLogContextName(name), true
	}
	return cat, true
}

func (ctx *SdlLoggingContext) dispatch(streamCtxName SdlLogContextName, logLevel log.LogLevel, msg string) {
	var interested []log.LogListener
	for listener, level := range ctx.listeners {
		if level >= logLevel || (level == log.Default && ctx.defaultListenerLevel <= logLevel) || level == log.All {
			interested = append(interested, listener)
		}
	}
	var stream *SdlLogStream
	if streamCtxName.Custom() {
		st, has := ctx.customStreams[string(streamCtxName)]
		if has {
			stream = st.(*SdlLogStream)
		}
	} else {
		stream = ctx.stdStreams[streamCtxName].(*SdlLogStream)
	}
	if stream != nil {
		for listener, level := range stream.listeners {
			if level >= logLevel || (level == log.Default && ctx.defaultListenerLevel <= logLevel) || level == log.All {
				interested = append(interested, listener)
			}
		}
	}
	if len(interested) > 0 {
		entry := &sdlLogEntry{
			timestamp: time.Now(),
			stream: streamCtxName,
			level: logLevel,
			msg: msg,
		}
		for _, l := range interested {
			go l.Receive(entry)
		}
	}
}

func (ctx *SdlLoggingContext) HasStream(key string) bool {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	lc := SdlLogContextName(key)
	if lc.Code() != C.SDL_LOG_CATEGORY_CUSTOM {
		_, has := ctx.customStreams[key]
		return has
	}
	return true
}

func (ctx *SdlLoggingContext) Stream(key string) (log.LogStream, bool) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	lc := SdlLogContextName(key)
	if lc.Code() == C.SDL_LOG_CATEGORY_CUSTOM {
		stream, has := ctx.customStreams[key]
		if has {
			return stream, true
		} 
		return nil, false
	}
	return ctx.stdStreams[SdlLogContextName(key)], true
}

func (ctx *SdlLoggingContext) DefaultLogLevel() log.LogLevel {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	return ctx.defaultLevel
}

func (ctx *SdlLoggingContext) SetDefaultLogLevel(level log.LogLevel) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.defaultLevel = level	
}

func (ctx *SdlLoggingContext) DefaultLogListenerLevel() log.LogLevel {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	return ctx.defaultListenerLevel
}

func (ctx *SdlLoggingContext) SetDefaultLogListenerLevel(level log.LogLevel) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.defaultListenerLevel = level
}

func (ctx *SdlLoggingContext) AddGlobalLogListener(logListener log.LogListener, level log.LogLevel) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.listeners[logListener] = level
}

func (ctx *SdlLoggingContext) RemoveGlobalLogListener(logListener log.LogListener) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	delete(ctx.listeners, logListener)
}

func (ctx *SdlLoggingContext) TracesByDefault() bool {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	return ctx.traces
}

func (ctx *SdlLoggingContext) SetTracesByDefault(traces bool) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.traces = traces
}

func (ctx *SdlLoggingContext) GlobalListeners() []log.LogListener {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	res := make([]log.LogListener, 0, len(ctx.listeners))
	for listener, _ := range ctx.listeners {
		res = append(res, listener)
	}
	return res
}

func (ctx *SdlLoggingContext) DebuggingEnabled() bool {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	return ctx.debugEnabled
}

func (ctx *SdlLoggingContext) EnableDebugging(val bool) {
	<-ctx.lock
	defer func() { ctx.lock <- true }()
	ctx.debugEnabled = val
}

func (ls *SdlLogStream) Log(level log.LogLevel, msg string) {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	pri := SdlLogPriorityForLogLevel(level)
	var cat int
	if stream, has := ls.ctx.stdStreams[SdlLogContextName(ls.name)]; has {
		cat = stream.(*SdlLogStream).categoryCode
	} else if stream, has := ls.ctx.customStreams[ls.name]; has {
		cat = stream.(*SdlLogStream).categoryCode
	} else {
		return
	}
	C.cgo_sdl_log_message(C.int(cat), C.SDL_LogPriority(pri), C.CString(msg))
}

func (ls *SdlLogStream) Logf(level log.LogLevel, format string, args ...interface{}) {
	ls.Log(level, fmt.Sprintf(format, args...))
}

func (ls *SdlLogStream) LogTrace(level log.LogLevel, msg string) {
	panic("SdlLogStream.LogTrace() unimplemented")
}

func (ls *SdlLogStream) LogTracef(level log.LogLevel, format string, args ...interface{}) {
	panic("SdlLogStream.LogTracef() unimplemented")
}

func (ls *SdlLogStream) Fatal(msg string) {
	ls.Log(log.FatalError, msg)
}

func (ls *SdlLogStream) Fatalf(format string, args ...interface{}) {
	ls.Logf(log.FatalError, format, args...)
}

func (ls *SdlLogStream) FatalTrace(msg string) {
	panic("SdlLogStream.FatalTrace() unimplemented")
}

func (ls *SdlLogStream) FatalTracef(format string, args ...interface{}) {
	panic("SdlLogStream.FatalTracef() unimplemented")
}

func (ls *SdlLogStream) Error(err error) {
	ls.Log(log.Error, err.Error())
}

func (ls *SdlLogStream) Errorf(err error, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	msg = fmt.Sprintf("%s: %s", err.Error(), msg)
	ls.Log(log.Error, msg)
}

func (ls *SdlLogStream) Warning(msg string) {
	ls.Log(log.Warning, msg)
}

func (ls *SdlLogStream) Warningf(format string, args ...interface{}) {
	ls.Warning(fmt.Sprintf(format, args...))
}

func (ls *SdlLogStream) WarningTrace(msg string) {
	panic("SdlLogStream.WarningTrace() unimplemented")
}

func (ls *SdlLogStream) WarningTracef(format string, args ...interface{}) {
	panic("SdlLogStream.WarningTracef() unimplemented")
}

func (ls *SdlLogStream) Info(msg string) {
	ls.Log(log.Info, msg)
}

func (ls *SdlLogStream) Infof(format string, args ...interface{}) {
	ls.Logf(log.Info, fmt.Sprintf(format, args...))
}

func (ls *SdlLogStream) InfoTrace(msg string) {
	panic("SdlLogStream.InfoTrace() unimplemented")
}

func (ls *SdlLogStream) InfoTracef(format string, args ...interface{}) {
	panic("SdlLogStream.InfoTracef() unimplemented")
}

func (ls *SdlLogStream) Debug(msg string) {
	ls.Log(log.Debug, msg)
}

func (ls *SdlLogStream) Debugf(format string, args ...interface{}) {
	ls.Log(log.Debug, fmt.Sprintf(format, args...))
}

func (ls *SdlLogStream) DebugTrace(msg string) {
	panic("SdlLogStream.DebugTrace() unimplemented")
}

func (ls *SdlLogStream) DebugTracef(format string, args ...interface{}) {
	panic("SdlLogStream.DebugTracef() unimplemented")
}

func (ls *SdlLogStream) Trace(msg string) {
	ls.Log(log.Trace, msg)
}

func (ls *SdlLogStream) Tracef(format string, args ...interface{}) {
	ls.Log(log.Trace, fmt.Sprintf(format, args...))
}

func (ls *SdlLogStream) Context() log.LoggingContext {
	return ls.ctx
}

func (ls *SdlLogStream) Name() string {
	return ls.name
}

func (ls *SdlLogStream) DefaultLogLevel() log.LogLevel {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	return ls.defaultLevel
}

func (ls *SdlLogStream) SetDefaultLogLevel(level log.LogLevel) {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	ls.defaultLevel = level
}

func (ls *SdlLogStream) DefaultLogListenerLevel() log.LogLevel {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	return ls.defaultListenerLevel
}

func (ls *SdlLogStream) SetDefaultLogListenerLevel(level log.LogLevel) {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	ls.defaultListenerLevel = level
}

func (ls *SdlLogStream) AddLogListener(logListener log.LogListener, level log.LogLevel) {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	ls.listeners[logListener] = level
}

func (ls *SdlLogStream) RemoveLogListener(logListener log.LogListener) {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	delete(ls.listeners, logListener)
}

func (ls *SdlLogStream) TracesByDefault() bool {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	return ls.traces
}

func (ls *SdlLogStream) SetTracesByDefault(traces bool) {
	<-ls.ctx.lock
	defer func() { ls.ctx.lock <- true }()
	ls.traces = traces
}

func (ls *SdlLogStream) IsActive() bool {
	return true
}

func (ls *SdlLogStream) Shutdown() {}


func (le *sdlLogEntry) LogTime() time.Time {
	return le.timestamp
}

func (le *sdlLogEntry) Stream() string {
	return string(le.stream)
}
func (le *sdlLogEntry) Level() log.LogLevel {
	return le.level
}

func (le *sdlLogEntry) Message() string {
	return le.msg
}

func (le *sdlLogEntry) HasAssociatedError() bool {
	return false
}

func (le *sdlLogEntry) AssociatedError() error {
	return nil
}

func (le *sdlLogEntry) HasTrace() bool {
	return false
}

func (le *sdlLogEntry) Trace() []*log.StackTraceEntry {
	return nil
}

// test shims to work around lack of cgo support for test cases
func test_SdlInit() {
	C.SDL_Init(C.SDL_INIT_EVERYTHING)
}

func test_SdlLog(msg string) {
	C.cgo_sdl_log(C.CString(msg))
}

func test_SdlQuit() {
	C.SDL_Quit()
}