package log

import (
	"runtime"
)

type StackTraceEntry struct {
	pc uintptr
	file string
	line int
	f *runtime.Func
}

func (ste *StackTraceEntry) Pc() uintptr {
	return ste.pc
}

func (ste *StackTraceEntry) File() string {
	return ste.file
}

func (ste *StackTraceEntry) Line() int {
	return ste.line
}

func (ste *StackTraceEntry) Function() *runtime.Func {
	return ste.f
}

func GenerateStackTrace() []*StackTraceEntry {
	trace := make([]*StackTraceEntry, 0, 16)
	for i := 1; i < 1000; i++ {
		pc, file, line, ok := runtime.Caller(2+i)
		if !ok {
			break
		}
		trace = append(trace, &StackTraceEntry{
			pc: pc,
			file: file,
			line: line,
		})
	}
	return trace 
}