package xdblog

import (
	"context"
	"fmt"
	"github.com/liweiming-nova/common/xlog"

	"time"

	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const (
	traceStr     = logger.Green + "%s " + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
	traceWarnStr = logger.Green + "%s " + logger.Yellow + "%s " + logger.Reset + logger.RedBold + "[%.3fms] " + logger.Yellow + "[rows:%v]" + logger.Magenta + " %s" + logger.Reset
)

type GormLog struct {
	SlowThreshold time.Duration
}

func NewGormLog() *GormLog {
	return &GormLog{
		SlowThreshold: 200 * time.Millisecond,
	}
}

func (m *GormLog) LogMode(LogLevel logger.LogLevel) logger.Interface {
	return m
}

func (m *GormLog) Info(ctx context.Context, msg string, data ...interface{}) {
	xlog.Infof(ctx, msg, data...)
}

func (m *GormLog) Warn(ctx context.Context, msg string, data ...interface{}) {
	xlog.Warnf(ctx, msg, data...)
}

func (m *GormLog) Error(ctx context.Context, msg string, data ...interface{}) {
	xlog.Errorf(ctx, msg, data...)
}

func (m *GormLog) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	file := utils.FileWithLineNum()
	if elapsed > m.SlowThreshold && m.SlowThreshold != 0 {
		slowLog := fmt.Sprintf("SLOW SQL >= %v", m.SlowThreshold)
		if rows == -1 {
			xlog.Infof(ctx, traceWarnStr, file, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			xlog.Infof(ctx, traceWarnStr, file, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	} else {
		if rows == -1 {
			xlog.Infof(ctx, traceStr, file, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			xlog.Infof(ctx, traceStr, file, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
	xlog.SLog(ctx, elapsed.Milliseconds(), file, err)
}
