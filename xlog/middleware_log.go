package xlog

import (
	"context"
	"fmt"
	"github.com/liweiming-nova/common/config"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

type MidLogConfig struct {
	// Grpc  *LogConfig `json:"log_grpc" yaml:"log_grpc" toml:"log_grpc"`
	Http *LogConfig `json:"log_http" yaml:"log_http" toml:"log_http"`
	// Kafka *LogConfig `json:"log_kafka" yaml:"log_kafka" toml:"log_kafka"`
	Sql *LogConfig `json:"log_mysql" yaml:"log_mysql" toml:"log_mysql"`
}

type MidLogger struct {
	// GrpcLogger  *zerolog.Logger
	HttpLogger *zerolog.Logger
	// KafkaLogger *zerolog.Logger
	SqlLogger *zerolog.Logger
}

var loggers = &MidLogger{}

var logFormat = "|level:%s|trace_id:%s|type:%s|rt:%d|success:%s|id:%s|error:%s"

func HLog(ctx context.Context, in bool, rt int64, path string, err error) {
	if loggers.HttpLogger == nil {
		return
	}
	t := "http-out"
	if in {
		t = "http-in"
	}
	log(loggers.HttpLogger, ctx, t, rt, path, err)
}

func SLog(ctx context.Context, rt int64, file string, err error) {
	if loggers.SqlLogger == nil {
		return
	}
	t := "sql"
	log(loggers.SqlLogger, ctx, t, rt, file, err)
}

func log(logger *zerolog.Logger, ctx context.Context, t string, rt int64, id string, err error) {
	if err == nil {
		logger.Info().Msgf(logFormat, "info", getTraceId(ctx), t, rt, "true", id, "")
	} else {
		logger.Error().Msgf(logFormat, "error", getTraceId(ctx), t, rt, "false", id, err.Error())
	}
}

func getTraceId(ctx context.Context) string {
	// todo
	return ""
}

func InitMLogger() {
	cfg := config.Get(&MidLogConfig{}).(*MidLogConfig)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"

	logTypes := []struct {
		config *LogConfig
		logger **zerolog.Logger
		name   string
	}{
		// {cfg.Grpc, &loggers.GrpcLogger, "log_grpc"},
		{cfg.Http, &loggers.HttpLogger, "log_http"},
		// {cfg.Kafka, &loggers.KafkaLogger, "log_kafka"},
		{cfg.Sql, &loggers.SqlLogger, "log_sql"},
	}

	for _, logType := range logTypes {
		if logType.config == nil {
			continue
		}
		writers, err := NewConsoleWriter(logType.config)
		if err != nil {
			panic(fmt.Errorf("%s日志配置异常:%w", logType.name, err))
		}
		newLogger := zerolog.New(zerolog.MultiLevelWriter(writers...)).With().Timestamp().Logger()
		*logType.logger = &newLogger
	}
}

func NewConsoleWriter(cfg *LogConfig) ([]io.Writer, error) {
	if cfg.LogFile == "" {
		return nil, fmt.Errorf("log_file不能为空")
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7
	}
	rotatingLog := &lumberjack.Logger{
		Filename: cfg.LogFile,
		MaxSize:  cfg.MaxSize,
		MaxAge:   cfg.MaxAge,
		Compress: cfg.Compress,
	}
	var writers []io.Writer
	filesWriter := &zerolog.ConsoleWriter{NoColor: true, TimeFormat: zerolog.TimeFieldFormat}
	filesWriter.Out = rotatingLog
	writers = append(writers, filesWriter)
	if cfg.Console {
		stdOutWriter := &zerolog.ConsoleWriter{NoColor: true, TimeFormat: zerolog.TimeFieldFormat}
		stdOutWriter.Out = os.Stdout
		writers = append(writers, stdOutWriter)
	}
	return writers, nil
}
