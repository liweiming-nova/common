package xlog

import (
	"context"
	"fmt"
	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/utils"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"runtime"
	"strings"
)

type funcTrace int

const (
	fieldTrace = "traceId"
	filePath   = "file"
	fieldFunc  = "funcId"

	funcTraceKey funcTrace = iota
)

const (
	Level      = "level"
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	Msg        = "msg"
	TraceId    = "trace_id"
	FuncId     = "func_id"
)

var orders = []string{Level, TraceId, filePath, Msg}

type ZeroLogger struct {
	Logger *zerolog.Logger

	enableCaller bool
	jsonFormat   bool
	newFormat    bool
}

func (m *ZeroLogger) Start() error {
	if m.Logger != nil {
		// 已经初始化
		return nil
	}

	cfg := config.Get(&logConfigs{}).(*logConfigs).Log
	m.Init(cfg)
	m.Infof(nil, "use zero logger config %+v", *cfg)
	DefaultLogger = m
	return nil
}

func (m *ZeroLogger) Stop() error {
	return nil
}

func (m *ZeroLogger) WithFuncTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, funcTraceKey, strings.Replace(utils.UUID(), "-", "", -1))
}

func (m *ZeroLogger) Init(cfg *LogConfig) {
	// 日志各字段名定义
	zerolog.TimestampFieldName = "time" // 时间
	zerolog.LevelFieldName = "level"    // 日志等级
	zerolog.MessageFieldName = "msg"    // 普通消息体
	zerolog.ErrorFieldName = "err"      // 错误
	zerolog.CallerFieldName = "cl"      // 调用者
	zerolog.ErrorStackFieldName = "st"  // 错误堆栈

	// 时间字段格式
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000" // Unix Time性能好，但没有 RFC3339 可读性好

	switch cfg.Level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		panic("invalid log level")
	}

	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7
	}

	// 先设置 enableCaller，这样后面的 partsOrder 配置才能正确
	m.enableCaller = cfg.EnableCaller

	var writers []io.Writer
	rotatingLog := &lumberjack.Logger{
		Filename: cfg.LogFile,
		MaxSize:  cfg.MaxSize,
		MaxAge:   cfg.MaxAge,
		Compress: cfg.Compress,
	}

	partsOrder := []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
	}
	fieldsExclude := []string{}

	if m.enableCaller {
		partsOrder = append(partsOrder, filePath)
		fieldsExclude = append(fieldsExclude, filePath)
	}

	m.newFormat = cfg.NewFormat
	if !m.newFormat {
		partsOrder = append(partsOrder, fieldTrace)
		fieldsExclude = append(fieldsExclude, fieldTrace)
	}

	partsOrder = append(partsOrder, zerolog.MessageFieldName)

	if cfg.JsonFormat {
		// json 格式
		m.jsonFormat = true
		writers = append(writers, rotatingLog)
		if cfg.Console {
			writers = append(writers, os.Stdout)
		}
	} else {
		// 控制台格式
		writers = append(writers, zerolog.ConsoleWriter{Out: rotatingLog, TimeFormat: zerolog.TimeFieldFormat, NoColor: true, PartsOrder: partsOrder, FieldsExclude: fieldsExclude})
		if cfg.Console {
			writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: zerolog.TimeFieldFormat, NoColor: true, PartsOrder: partsOrder, FieldsExclude: fieldsExclude})
		}
	}

	c := zerolog.New(zerolog.MultiLevelWriter(writers...)).With().Timestamp()
	l := c.Logger()
	m.Logger = &l
}

func (m *ZeroLogger) Debug(ctx context.Context, msg string) {
	if m.newFormat {
		m.Logger.Debug().Msg(m.addExtInfo(ctx, DebugLevel, msg))
		return
	}
	e := m.Logger.Debug()
	m.addExt(ctx, e)
	e.Msg(msg)
}

func (m *ZeroLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if m.newFormat {
		m.Debug(ctx, fmt.Sprintf(format, args...))
		return
	}
	e := m.Logger.Debug()
	m.addExt(ctx, e)
	e.Msgf(format, args...)
}

func (m *ZeroLogger) Info(ctx context.Context, msg string) {
	if m.newFormat {
		m.Logger.Info().Msg(m.addExtInfo(ctx, InfoLevel, msg))
		return
	}
	e := m.Logger.Info()
	m.addExt(ctx, e)
	e.Msg(msg)
}

func (m *ZeroLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	if m.newFormat {
		m.Info(ctx, fmt.Sprintf(format, args...))
		return
	}
	e := m.Logger.Info()
	m.addExt(ctx, e)
	e.Msgf(format, args...)
}

func (m *ZeroLogger) Error(ctx context.Context, msg string) {
	if m.newFormat {
		m.Logger.Error().Msg(m.addExtInfo(ctx, ErrorLevel, msg))
		return
	}
	e := m.Logger.Error()
	m.addExt(ctx, e)
	e.Msg(msg)
}

func (m *ZeroLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if m.newFormat {
		m.Error(ctx, fmt.Sprintf(format, args...))
		return
	}
	e := m.Logger.Error()
	m.addExt(ctx, e)
	e.Msgf(format, args...)
}

func (m *ZeroLogger) Warn(ctx context.Context, msg string) {
	if m.newFormat {
		m.Logger.Warn().Msg(m.addExtInfo(ctx, WarnLevel, msg))
		return
	}
	e := m.Logger.Warn()
	m.addExt(ctx, e)
	e.Msg(msg)
}

func (m *ZeroLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if m.newFormat {
		m.Warn(ctx, fmt.Sprintf(format, args...))
		return
	}
	e := m.Logger.Warn()
	m.addExt(ctx, e)
	e.Msgf(format, args...)
}

func (m *ZeroLogger) addExtInfo(ctx context.Context, level string, msg string) string {
	kv := make(map[string]string)
	kv[Level] = level
	kv[TraceId] = m.getTraceId(ctx)
	kv[filePath] = callerInfoV2()
	kv[Msg] = msg
	var res = ""
	for _, k := range orders {
		if v, ok := kv[k]; ok {
			res += fmt.Sprintf("|%s:%s", k, v)
		}
	}
	if ctx != nil {
		if v, ok := ctx.Value(funcTraceKey).(string); ok {
			res += fmt.Sprintf(" [%s]=%s", FuncId, v)
		}
	}
	return res
}

func (m *ZeroLogger) getTraceId(ctx context.Context) string {

	return ""
}

func (m *ZeroLogger) getTraceStr(ctx context.Context) string {

	if m.jsonFormat {
		return ""
	} else {
		return "[]"
	}
}

func (m *ZeroLogger) addExt(ctx context.Context, event *zerolog.Event) {
	// 按照期望的顺序添加字段：先文件路径，再traceId
	if m.enableCaller {
		event.Str(filePath, callerInfo())
	}
	event.Str(fieldTrace, m.getTraceStr(ctx))

	if ctx == nil {
		return
	}

	if v, ok := ctx.Value(funcTraceKey).(string); ok {
		event.Str(fieldFunc, v)
	}
}

func callerInfo() string {
	_, file, line, ok := runtime.Caller(4)
	if ok {
		return fmt.Sprintf("[%s:%d]", file, line)
	}
	return ""
}

func callerInfoV2() string {
	_, file, line, ok := runtime.Caller(5)
	if ok {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return ""
}

func NewZeroLogger() *ZeroLogger {
	return &ZeroLogger{}
}
