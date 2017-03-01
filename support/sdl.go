//+build sdl

package support

import (
	"unsafe"
	"github.com/dtromb/log"
	"reflect"
)

/*
	#cgo pkg-config: sdl2
	#include <SDL_log.h>
	
	extern sdlLogOutputDispatch(char *userdata, int category, SDL_LogPriority pri, char *msg);
	
	void cgo_sdl_log_output_dispatch(void *userdata, int category, SDL_LogPriority pri, const char *msg) {
		sdlLogOutputDispatch((char*)userdata, category, pri, (char*)msg);
	}
	

*/
import "C"


/*
void SDL_LogOutputFunction(void*           userdata,
                           int             category,
                           SDL_LogPriority priority,
                           const char*     message)
*/

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
	
}

type SdlLogStream struct {
	ctx *SdlLoggingContext
	name string
}

type SdlLogUserdata struct {
	lock chan bool
	contexts map[*SdlLoggingContext]bool
}

var global_SdlLogUserdata *SdlLogUserdata = &SdlLogUserdata{
	lock: make(chan bool, 1),
	contexts: make(map[*SdlLoggingContext]bool),
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
		}
		ctx.stdStreams[key] = nls
	}
	ptr := reflect.ValueOf(global_SdlLogUserdata).Pointer()
	<-global_SdlLogUserdata.lock
	defer func() { global_SdlLogUserdata.lock <- true }()
	C.SDL_LogSetOutputFunction((*[0]byte)(C.cgo_sdl_log_output_dispatch), unsafe.Pointer(ptr))
	global_SdlLogUserdata.contexts[ctx] = true
	ctx.lock <- true
	return ctx
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
	panic("SdlLoggingContext.dispatch() unimplemented")
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

func (ls *SdlLogStream) Log(level log.LogLevel, msg string)
func (ls *SdlLogStream) Logf(level log.LogLevel, format string, args ...interface{})
func (ls *SdlLogStream) LogTrace(level log.LogLevel, msg string)
func (ls *SdlLogStream) LogTracef(level log.LogLevel, format string, args ...interface{})
func (ls *SdlLogStream) Fatal(msg string)
func (ls *SdlLogStream) Fatalf(format string, args ...interface{})
func (ls *SdlLogStream) FatalTrace(msg string)
func (ls *SdlLogStream) FatalTracef(format string, args ...interface{})
func (ls *SdlLogStream) Error(err error)
func (ls *SdlLogStream) Errorf(err error, format string, args ...interface{})
func (ls *SdlLogStream) Warning(msg string)
func (ls *SdlLogStream) Warningf(format string, args ...interface{})
func (ls *SdlLogStream) WarningTrace(msg string)
func (ls *SdlLogStream) WarningTracef(format string, args ...interface{})
func (ls *SdlLogStream) Info(msg string)
func (ls *SdlLogStream) Infof(format string, args ...interface{})
func (ls *SdlLogStream) InfoTrace(msg string)
func (ls *SdlLogStream) InfoTracef(format string, args ...interface{})
func (ls *SdlLogStream) Debug(msg string) 
func (ls *SdlLogStream) Debugf(format string, args ...interface{})
func (ls *SdlLogStream) DebugTrace(msg string) 
func (ls *SdlLogStream) DebugTracef(format string, args ...interface{})
func (ls *SdlLogStream) Trace(msg string)
func (ls *SdlLogStream) Tracef(format string, args ...interface{})
func (ls *SdlLogStream) Context() log.LoggingContext
func (ls *SdlLogStream) Name() string
func (ls *SdlLogStream) DefaultLogLevel() log.LogLevel
func (ls *SdlLogStream) SetDefaultLogLevel(level log.LogLevel)
func (ls *SdlLogStream) DefaultLogListenerLevel() log.LogLevel
func (ls *SdlLogStream) SetDefaultLogListenerLevel(level log.LogLevel)
func (ls *SdlLogStream) AddLogListener(logListener log.LogListener, level log.LogLevel)
func (ls *SdlLogStream) RemoveLogListener(logListener log.LogListener)
func (ls *SdlLogStream) TracesByDefault() bool
func (ls *SdlLogStream) SetTracesByDefault(traces bool)
func (ls *SdlLogStream) IsActive() bool
func (ls *SdlLogStream) Shutdown()