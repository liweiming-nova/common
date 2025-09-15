// server/grpc_log_interceptor.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liweiming-nova/common/xlog"
	"google.golang.org/grpc"
)

// LogInterceptor 是一个 gRPC 拦截器，用于记录请求和响应的日志
type LogInterceptor struct {
	EnableRequestLog  bool
	EnableResponseLog bool
	EnableErrorLog    bool
	LogRequestArgs    bool
	LogResponseResult bool
	MaxLogLength      int
}

// NewLogInterceptor 创建一个新的日志插件（配置方式不变）
func NewLogInterceptor(options ...LogInterceptorOption) *LogInterceptor {
	plugin := &LogInterceptor{
		EnableRequestLog:  true,
		EnableResponseLog: true,
		EnableErrorLog:    true,
		LogRequestArgs:    true,
		LogResponseResult: true,
		MaxLogLength:      1024,
	}

	for _, option := range options {
		option(plugin)
	}

	return plugin
}

// LogInterceptorOption 保持不变
type LogInterceptorOption func(*LogInterceptor)

func WithEnableRequestLog(enable bool) LogInterceptorOption {
	return func(p *LogInterceptor) { p.EnableRequestLog = enable }
}

func WithEnableResponseLog(enable bool) LogInterceptorOption {
	return func(p *LogInterceptor) { p.EnableResponseLog = enable }
}

func WithEnableErrorLog(enable bool) LogInterceptorOption {
	return func(p *LogInterceptor) { p.EnableErrorLog = enable }
}

func WithLogRequestArgs(enable bool) LogInterceptorOption {
	return func(p *LogInterceptor) { p.LogRequestArgs = enable }
}

func WithLogResponseResult(enable bool) LogInterceptorOption {
	return func(p *LogInterceptor) { p.LogResponseResult = enable }
}

func WithMaxLogLength(length int) LogInterceptorOption {
	return func(p *LogInterceptor) { p.MaxLogLength = length }
}

// UnaryServerInterceptor 实现 gRPC Unary 拦截器
func (p *LogInterceptor) UnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {

	startTime := time.Now()

	serviceName, methodName := parseFullMethod(info.FullMethod)

	// 记录请求日志
	if p.EnableRequestLog {
		logMsg := fmt.Sprintf("gRPC Request - Service: %s, Method: %s", serviceName, methodName)
		if p.LogRequestArgs && req != nil {
			argsStr := p.safeMarshal(req)
			logMsg += fmt.Sprintf(", Args: %s", argsStr)
		}
		xlog.Infof(ctx, logMsg)
	}

	// 调用实际 handler
	resp, err = handler(ctx, req)

	// 记录响应日志
	duration := time.Since(startTime)
	logMsg := fmt.Sprintf("gRPC Response - Service: %s, Method: %s, Duration: %v", serviceName, methodName, duration)

	if err != nil {
		if p.EnableErrorLog {
			xlog.Errorf(ctx, "%s, Error: %v", logMsg, err)
		}
	} else if p.EnableResponseLog {
		if p.LogResponseResult && resp != nil {
			resultStr := p.safeMarshal(resp)
			logMsg += fmt.Sprintf(", Result: %s", resultStr)
		}
		xlog.Infof(ctx, logMsg)
	}

	return resp, err
}

// parseFullMethod 从 /package.Service/Method 解析出 Service 和 Method
func parseFullMethod(fullMethod string) (service, method string) {
	parts := strings.Split(fullMethod, "/")
	if len(parts) != 3 {
		return "unknown", "unknown"
	}
	// parts[1] = "package.Service"
	service = parts[1]
	method = parts[2]
	// 可选：去掉包名，只保留 Service 名
	if idx := strings.LastIndex(service, "."); idx != -1 {
		service = service[idx+1:]
	}
	return service, method
}

// safeMarshal 安全地序列化对象为JSON字符串（保持不变）
func (p *LogInterceptor) safeMarshal(v interface{}) string {
	if v == nil {
		return ""
	}

	data, err := json.Marshal(v)
	if err != nil {
		return p.truncateString(fmt.Sprintf("%+v", v))
	}

	return p.truncateString(string(data))
}

// truncateString 截断字符串（保持不变）
func (p *LogInterceptor) truncateString(s string) string {
	if len(s) <= p.MaxLogLength {
		return s
	}
	return s[:p.MaxLogLength] + "..."
}
