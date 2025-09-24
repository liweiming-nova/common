package server

import (
	"errors"
	"fmt"
	"github.com/liweiming-nova/common/grpcx/register"
	"github.com/liweiming-nova/common/utils"
	"google.golang.org/grpc"
	"net"
	"strings"
)

type GrpcServer struct {
	cfg      *GrpcConfig
	server   *grpc.Server
	register register.Register
	listener net.Listener
	name     string
	addr     string
	started  bool
}

func NewGrpcServer(cfg *GrpcConfig, name string, registerFunc func(*grpc.Server), interceptors ...grpc.UnaryServerInterceptor) (r *GrpcServer, err error) {
	r = &GrpcServer{
		cfg:  cfg,
		name: name,
	}
	if r.name == "" {
		r.name = "default"
	}

	if cfg.Register == "" {
		cfg.Register = "etcd"
	}

	var lisIp, lisPort string
	// 尝试获取本机ip 默认值
	if lisIp, err = utils.GetLocalIP(); err != nil || len(lisIp) == 0 {
		err = fmt.Errorf("ip parse fail, %s", err)
	}
	// 通过配置文件获取
	if err == nil {
		if t := strings.Split(cfg.DialAddr, ":"); len(t) != 2 || len(t[1]) == 0 {
			err = fmt.Errorf("port parse fail")
		} else {
			lisPort = t[1]
		}
	}
	if err != nil {
		return
	}

	addr := fmt.Sprintf("%s:%s", lisIp, lisPort)
	r.addr = addr

	switch cfg.Register {
	case "etcd":
		r.register = register.NewEtcdRegister()
	}
	// 添加日志插件
	if cfg.EnableLogPlugin {
		logInterceptor := NewLogInterceptor(
			WithEnableRequestLog(cfg.EnableRequestLog),
			WithEnableResponseLog(cfg.EnableResponseLog),
			WithEnableErrorLog(cfg.EnableErrorLog),
			WithLogRequestArgs(cfg.LogRequestArgs),
			WithLogResponseResult(cfg.LogResponseResult),
			WithMaxLogLength(cfg.LogMaxLength),
		)
		interceptors = append(interceptors, logInterceptor.UnaryServerInterceptor)
	}

	// 使用 ChainUnaryInterceptor 支持多个拦截器
	var serverOptions []grpc.ServerOption
	if len(interceptors) > 0 {
		serverOptions = append(serverOptions, grpc.ChainUnaryInterceptor(interceptors...))
	}

	// 创建 gRPC Server
	r.server = grpc.NewServer(serverOptions...)

	registerFunc(r.server)

	return
}

// Start 启动 gRPC 服务（不带额外注册）
func (s *GrpcServer) Start() error {
	if s.started {
		return fmt.Errorf("server %s already started", s.name)
	}

	// 监听端口
	listener, err := net.Listen("tcp", s.cfg.DialAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.DialAddr, err)
	}
	s.listener = listener

	s.started = true

	// 异步启动，不阻塞
	go func() {
		fmt.Printf("🚀 gRPC Server [%s] started on %s\n", s.name, s.cfg.DialAddr)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			fmt.Printf("❌ gRPC Server [%s] stopped with error: %v\n", s.name, err)
		}
	}()

	err = s.register.Register(s.name, s.addr, make(map[string]string))
	if err != nil {
		return fmt.Errorf("failed to register gRPC Server [%s]: %w", s.name, err)
	}

	return nil
}

func (s *GrpcServer) Stop() error {
	err := s.register.Unregister(s.name, s.addr)
	if err != nil {
		return err
	}
	s.server.Stop()

	return nil
}
