package log

import (
	"io"
	"fmt"
)

type LogListener interface {
	Name() string
	Receive(entry LogEntry)
	Close() error
}

type FormattingLogListener interface {
	LogListener
	Formatter() LogEntryFormatter
}

type StandardLogFormatterFlags uint16 
const (
	Zero					StandardLogFormatterFlags = 1 << iota
	PrintTime		
	PrintFileLine      
	PrintErrorMsg
	PrintStackTrace 	
	PrintStreamName
	PrintLevel
	PrintMessage
	PrintNewline
	PrintColor
)

type BaseColor uint8
const (
	Black			BaseColor  = iota	
	Red	
	Green	
	Yellow	
	Blue	
	Magenta	
	Cyan
	White
	DefaultColor
)

type ColorPrefix string

func MakeColorPrefix(bg BaseColor, fg BaseColor, fgBright bool) ColorPrefix {
	var buf []byte
	buf = append(buf, []byte("\x1b[")...)
	if fgBright {
		buf = append(buf, []byte(fmt.Sprintf("%d;1", 30+fg))...)
	} else {
		buf = append(buf, []byte(fmt.Sprintf("%d;2", 30+fg))...)
	}
	if bg != DefaultColor {
		buf = append(buf, []byte(fmt.Sprintf(";%d", 40+bg))...)
	}
	buf = append(buf, 'm')
	return ColorPrefix(buf)
}

type StandardLogFormatter interface {
	LogEntryFormatter
	SetFlags(flags StandardLogFormatterFlags)
	ClearFlags(flags StandardLogFormatterFlags)
	TimeFormat() string
	SetTimeFormat(format string)
	FieldSeparator() string
	SetFieldSepartor(fsep string)
	Indent() string
	SetIndent(indent string)
	GetLevelColorPrefix(level LogLevel) ColorPrefix
	SetLevelColorPrefix(level LogLevel, prefix ColorPrefix) 
}

///

type stdLogEntryFormatter struct {
	flags StandardLogFormatterFlags
	timeFormat string
	sep string
	indent string
	colorPrefixes map[LogLevel]ColorPrefix
}

func NewLogEntryFormatter() StandardLogFormatter {
	slf := &stdLogEntryFormatter{
		flags: PrintTime | PrintStreamName | PrintLevel | PrintMessage | 
		       PrintFileLine | PrintErrorMsg | PrintNewline | PrintStackTrace,
		timeFormat: "01/02/06 15:04:05.000",
		sep: " | ",
		indent: "   ",
		colorPrefixes: make(map[LogLevel]ColorPrefix),
	}
	slf.SetLevelColorPrefix(Debug, MakeColorPrefix(DefaultColor, White, false))
	slf.SetLevelColorPrefix(Trace, MakeColorPrefix(DefaultColor, White, false))
	slf.SetLevelColorPrefix(Info, MakeColorPrefix(DefaultColor, White, true))
	slf.SetLevelColorPrefix(Warning, MakeColorPrefix(DefaultColor, Yellow, true))
	slf.SetLevelColorPrefix(Error, MakeColorPrefix(DefaultColor, Red, true))
	slf.SetLevelColorPrefix(FatalError, MakeColorPrefix(Red, Yellow, true))
	return slf
}


func (lef *stdLogEntryFormatter) Format(entry LogEntry) string {
	var buf []byte
	fc := 0
	cp := lef.GetLevelColorPrefix(entry.Level())
	fsep := func() { 
		if lef.flags & PrintColor != 0 {
			buf = append(buf, []byte{0x1B,0x00,0x5B,0x33,0x39,0x3B,0x34,0x39,0x3B,0x32,0x32,0x6D}...)
		}
		if fc > 0 {
			buf = append(buf, []byte(lef.sep)...)
			if lef.flags & PrintColor != 0 {
				buf = append(buf, []byte(cp)...)
			}
		}
		fc++
	}
	if lef.flags & PrintColor != 0 {
		buf = append(buf, []byte(cp)...)
	}
	if lef.flags & PrintTime != 0 {
		fsep()
		buf = append(buf, []byte(entry.LogTime().Format(lef.timeFormat))...)
	}
	if lef.flags & PrintStreamName != 0 {
		fsep()
		buf = append(buf, []byte(entry.Stream())...)
	}
	if lef.flags & PrintLevel != 0 {
		fsep()
		buf = append(buf, []byte(entry.Level().String())...)
	}
	if lef.flags & PrintMessage != 0{
		fsep()
		buf = append(buf, []byte(entry.Message())...)
	}
	if entry.HasTrace() && lef.flags & PrintFileLine != 0 {
		traceFrame := entry.Trace()[0]
		fsep()
		buf = append(buf, fmt.Sprintf("%s:%d", traceFrame.File(), traceFrame.Line())...)
	}
	if lef.flags & PrintErrorMsg != 0 && entry.HasAssociatedError() {
		if lef.flags & PrintNewline != 0 {
			if lef.flags & PrintColor != 0 {
				buf = append(buf, []byte{0x1B,0x00,0x5B,0x33,0x39,0x3B,0x34,0x39,0x6D}...)
			}
			buf = append(buf, '\n')
			buf = append(buf, []byte(lef.indent)...)
			buf = append(buf, []byte(entry.AssociatedError().Error())...)
		} else {
			fsep()
			buf = append(buf, []byte(entry.AssociatedError().Error())...)
		}
	}
	if lef.flags & PrintStackTrace != 0 && entry.HasTrace() {
		for i, frame := range entry.Trace() {
			buf = append(buf, fmt.Sprintf("\n%s[%d] %s:%d in %s()", lef.indent, i, frame.File(), frame.Line(), frame.Function().Name())...)
		}
	}
	if lef.flags & PrintNewline != 0 {
		if lef.flags & PrintColor != 0 {
			buf = append(buf, []byte{0x1B,0x00,0x5B,0x33,0x39,0x3B,0x34,0x39,0x6D}...)
		}
		buf = append(buf, '\n')
	}
	if lef.flags & PrintColor != 0 {
		buf = append(buf, []byte{0x1B,0x00,0x5B,0x33,0x39,0x3B,0x34,0x39,0x3B,0x32,0x32,0x6D}...)
	}
	buf = append(buf, ' ')
	return string(buf)
}

func (lef *stdLogEntryFormatter) SetFlags(flags StandardLogFormatterFlags) {
	lef.flags = lef.flags | flags
}

func (lef *stdLogEntryFormatter) ClearFlags(flags StandardLogFormatterFlags) {
	lef.flags = lef.flags & ^flags
}

func (lef *stdLogEntryFormatter) TimeFormat() string {
	return lef.timeFormat
}

func (lef *stdLogEntryFormatter) SetTimeFormat(format string) {
	lef.timeFormat = format
}

func (lef *stdLogEntryFormatter) FieldSeparator() string {
	return lef.sep
}

func (lef *stdLogEntryFormatter) SetFieldSepartor(fsep string) {
	lef.sep = fsep

}
func (lef *stdLogEntryFormatter) Indent() string {
	return lef.indent
}

func (lef *stdLogEntryFormatter) SetIndent(indent string) {
	lef.indent = indent
}

func (lef *stdLogEntryFormatter) GetLevelColorPrefix(level LogLevel) ColorPrefix {
	cp, has := lef.colorPrefixes[level]
	if !has {
		return MakeColorPrefix(DefaultColor, DefaultColor, true)
	}
	return cp
}

func (lef *stdLogEntryFormatter) SetLevelColorPrefix(level LogLevel, prefix ColorPrefix) {
	lef.colorPrefixes[level] = prefix
}

type writerLogger struct {
	formatter LogEntryFormatter
	out io.Writer
	name string
}

func NewWriterLogger(name string, writer io.Writer, formatter LogEntryFormatter) LogListener {
	return &writerLogger{
		formatter: formatter,
		out: writer,
		name: name,
	}
}

func (wl *writerLogger) Receive(entry LogEntry) {
	str := wl.formatter.Format(entry)
	wl.out.Write([]byte(str))
}

func (wl *writerLogger) Name() string {
	return wl.name
}

func (wl *writerLogger) Close() error {
	if wc, ok := wl.out.(io.WriteCloser); ok {
		return wc.Close()
	}
	return nil
}

func (wl *writerLogger) Formatter() LogEntryFormatter {
	return wl.formatter
}