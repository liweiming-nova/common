package plugins

import (
	"github.com/liweiming-nova/common/grpcx/server"
	"google.golang.org/grpc"
)

type Options func(plugins *GRPCPlugins)

type GRPCPlugins struct {
	interceptor  []grpc.UnaryServerInterceptor //拦截器
	registerFunc func(*grpc.Server)
	name         string
}

func WithInterceptors(interceptors ...grpc.UnaryServerInterceptor) Options {
	return func(plugins *GRPCPlugins) {
		plugins.interceptor = interceptors
	}
}

func WithRegisterFunc(re func(*grpc.Server)) Options {
	return func(plugins *GRPCPlugins) {
		plugins.registerFunc = re
	}
}

func WithGRPCName(name string) Options {
	return func(plugins *GRPCPlugins) {
		plugins.name = name
	}
}

func NewGRPCPlugins(options ...Options) *GRPCPlugins {
	plugins := &GRPCPlugins{}
	for _, option := range options {
		option(plugins)
	}
	return plugins
}

func (plugins *GRPCPlugins) Start(ctx *PluginContext) error {
	if plugins.registerFunc != nil {
		panic("grpc registerFunc is nil")
	}
	return server.StartServer(plugins.name, plugins.registerFunc, plugins.interceptor)
}

func (plugins *GRPCPlugins) Stop() error {
	return server.StopServer(plugins.name)
}

func (plugins *GRPCPlugins) BeforeStart(ctx *PluginContext) error {
	return nil
}
