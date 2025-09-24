package xlog

import "context"

var (
	DefaultLogger xLogger
)

func init() {
	// 默认
	zeroLogger := NewZeroLogger()
	zeroLogger.Init(&LogConfig{
		Level:        "info",
		LogFile:      "runtime/logs",
		Console:      true,
		MaxSize:      100,
		MaxAge:       7,
		Compress:     true,
		JsonFormat:   false,
		EnableCaller: false,
		NewFormat:    false,
	})
	DefaultLogger = zeroLogger
}

type LogConfig struct {
	Level        string `json:"level" yaml:"level" toml:"level"`                              // 日志打印级别
	LogFile      string `json:"log_file" yaml:"log_file" toml:"log_file" validate:"required"` // 日志路径
	Console      bool   `json:"console" yaml:"console" toml:"console"`                        // 是否同步输出到控制台
	MaxSize      int    `json:"max_size" yaml:"max_size" toml:"max_size"`                     // 日志文件最大大小，单位为M，默认100M
	MaxAge       int    `json:"max_age" yaml:"max_age" toml:"max_age"`                        // 日志留存最大天数，默认7天
	Compress     bool   `json:"compress" yaml:"compress" toml:"compress"`                     // 是否对日志文件压缩
	JsonFormat   bool   `json:"json_format" yaml:"json_format" toml:"json_format"`            // json 格式
	EnableCaller bool   `json:"enable_caller" yaml:"enable_caller" toml:"enable_caller"`      // 是否启用调用者信息（文件路径和行号），默认false
	NewFormat    bool   `json:"new_format" yaml:"new_format" toml:"new_format"`               // 启用新格式打印
}

type logConfigs struct {
	Log *LogConfig `json:"log" yaml:"log" toml:"log" mapstructure:"log"`
}

type xLogger interface {
	Debugf(ctx context.Context, format string, args ...interface{})
	Debug(ctx context.Context, msg string)
	Infof(ctx context.Context, format string, args ...interface{})
	Info(ctx context.Context, msg string)
	Errorf(ctx context.Context, format string, args ...interface{})
	Error(ctx context.Context, msg string)
	Warnf(ctx context.Context, format string, args ...interface{})
	Warn(ctx context.Context, msg string)

	WithFuncTrace(ctx context.Context) context.Context
	Start() error
	Stop() error
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	DefaultLogger.Debugf(ctx, format, args...)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	// traceId := htrace.GetTraceId(ctx)
	// seelog.Infof("["+traceId+"] "+format, args...)  // TODO
	DefaultLogger.Infof(ctx, format, args...)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	DefaultLogger.Errorf(ctx, format, args...)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	DefaultLogger.Warnf(ctx, format, args...)
}

// === 普通日志（无格式化）===

func Debug(ctx context.Context, msg string) {
	DefaultLogger.Debug(ctx, msg)
}

func Info(ctx context.Context, msg string) {
	DefaultLogger.Info(ctx, msg)
}

func Warn(ctx context.Context, msg string) {
	DefaultLogger.Warn(ctx, msg)
}

func Error(ctx context.Context, msg string) {
	DefaultLogger.Error(ctx, msg)
}

func Start() error {
	return DefaultLogger.Start()
}

func Stop() error {
	// 退出前关闭日志
	return DefaultLogger.Stop()
}

func WithFuncTrace(ctx context.Context) context.Context {
	return DefaultLogger.WithFuncTrace(ctx)
}
