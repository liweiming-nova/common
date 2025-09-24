package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/config/options"
	"github.com/liweiming-nova/common/grpcx/discovery"
	"github.com/liweiming-nova/common/grpcx/instance"
	"github.com/liweiming-nova/common/xlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type rpcConfig struct {
	Rpc *struct {
		Cfgs map[string]*Cfg `toml:"client"`
	} `toml:"rpc"`
}

type Cfg struct {
	DialTimeout        time.Duration `toml:"dial_timeout"`
	DialFailMode       string        `toml:"fail_mode"`
	DialSelectMode     string        `toml:"select_mode"`
	DialConnectTimeout time.Duration `toml:"dial_timeout"`
	ServiceName        string        `toml:"service_name"`
	// pool
	PoolMaxActive int `toml:"pool_max_active"` // 最大活跃连接数
}

// GrpcClientPool 是一个固定大小的 gRPC 客户端连接池
// 使用 round-robin 算法选择客户端
type GrpcClientPool struct {
	count   uint64
	index   uint64
	clients []*grpc.ClientConn
	mu      sync.RWMutex

	clientTimeout time.Duration
	// todo  节点选择和失败重试策略
	failMode   string
	selectMode string

	// 服务发现
	discovery   discovery.ServiceDiscovery
	serviceName string
	addrIdx     uint64

	// Watch 支持（可选）
	watchCh chan []*discovery.KVPair
}

// NewGrpcClientPool 创建一个固定大小的 gRPC 客户端池
func NewGrpcClientPool(count int, cfg *Cfg, discovery discovery.ServiceDiscovery) (*GrpcClientPool, error) {
	if count <= 0 {
		count = 10
	}

	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	pool := &GrpcClientPool{
		count:         uint64(count),
		clients:       make([]*grpc.ClientConn, count),
		failMode:      cfg.DialFailMode,
		selectMode:    cfg.DialSelectMode,
		clientTimeout: cfg.DialTimeout,
		discovery:     discovery,
		serviceName:   cfg.ServiceName,
	}

	// 预创建所有客户端连接
	for i := 0; i < count; i++ {
		conn, err := pool.newClientConn()
		if err != nil {
			// 清理已创建的连接
			pool.Close()
			return nil, fmt.Errorf("failed to create client %d: %w", i, err)
		}
		pool.clients[i] = conn
	}

	return pool, nil
}

// Close 关闭所有客户端连接
func (p *GrpcClientPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.clients {
		if conn != nil {
			conn.Close()
		}
	}
	p.clients = nil
}

// newClientConn 创建单个 grpc.ClientConn（简化版，可扩展服务发现）
func (p *GrpcClientPool) newClientConn() (*grpc.ClientConn, error) {
	var target string

	if p.discovery == nil {
		return nil, errors.New("discovery is nil")
	}

	kvPairs := p.discovery.GetServices()
	if len(kvPairs) == 0 {
		return nil, errors.New("discovery has no services")
	}
	// 从服务发现选一个 todo
	addrIndex := atomic.AddUint64(&p.addrIdx, 1) % uint64(len(kvPairs))
	value := kvPairs[addrIndex].Value
	// 统一通过公共方法提取地址
	target = instance.ExtractAddress(value)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	// 使用带超时的 DialContext
	ctxDial, cancelDial := context.WithTimeout(context.Background(), p.clientTimeout)
	defer cancelDial()
	conn, err := grpc.DialContext(ctxDial, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for %s: %w", target, err)
	}

	// 使用阻塞拨号已由 DialContext 控制超时，若失败会返回 err

	return conn, nil
}

// Get 返回一个客户端（不移除，不关闭，线程安全）
// 使用 round-robin 策略
func (p *GrpcClientPool) Get() *grpc.ClientConn {
	if p == nil || len(p.clients) == 0 {
		return nil
	}

	i := atomic.AddUint64(&p.index, 1)
	picked := int(i % p.count)
	return p.clients[picked]
}

// Call 通过连接池调用 gRPC 方法
func (p *GrpcClientPool) Call(ctx context.Context, method string, req proto.Message, resp proto.Message) error {
	client := p.Get()
	if client == nil {
		return fmt.Errorf("no available client in pool")
	}

	// 规范化方法名：
	// - 若已是全限定名（以 / 开头），直接使用
	// - 否则自动拼接为 /<ServiceName>/<Method>
	normalized := method
	if !strings.HasPrefix(method, "/") {
		svc := strings.TrimSpace(p.serviceName)
		if svc == "" {
			return fmt.Errorf("service name is empty, cannot build full method name for %q", method)
		}
		// 确保服务名不带前导斜杠
		svc = strings.TrimPrefix(svc, "/")
		normalized = "/" + svc + "/" + strings.TrimPrefix(method, "/")
	}

	// 可选：添加 metadata、超时控制、重试逻辑
	return client.Invoke(ctx, normalized, req, resp)
}

var (
	once  sync.Once
	lock  sync.RWMutex
	pools map[string]*GrpcClientPool
)

func init() {
	pools = map[string]*GrpcClientPool{}
}

func Call(ctx context.Context, name string, method string, req proto.Message, resp proto.Message) (err error) {
	var cli *GrpcClientPool
	cli, err = SafeClient(name)
	if err == nil {
		err = cli.Call(ctx, method, req, resp)
	}
	return
}

func SafeClient(name string) (r *GrpcClientPool, err error) {
	r, err = SafePool(name)
	if err != nil {
		return
	}
	return
}

func SafePool(name string) (r *GrpcClientPool, err error) {
	return getPool(name)
}

func getPool(name string) (r *GrpcClientPool, err error) {
	lock.RLock()
	r = pools[name]
	lock.RUnlock()
	if r == nil {
		r, err = addPool(name)
	}
	return
}
func addPool(name string) (r *GrpcClientPool, err error) {
	var cfg *Cfg
	if cfg, err = loadCfg(name); err != nil {
		return
	}
	r, err = NewRpcClientPool(name, cfg)

	lock.Lock()
	pools[name] = r
	lock.Unlock()
	return
}

func loadCfg(name string) (r *Cfg, err error) {
	var cfgs map[string]*Cfg
	if cfgs, err = loadCfgs(); err != nil {
		return
	}
	if r = cfgs[name]; r == nil {
		err = fmt.Errorf("rpcx#%s not configed", name)
		return
	}
	return
}

func loadCfgs() (r map[string]*Cfg, err error) {
	r = map[string]*Cfg{}

	once.Do(func() {
		config.Get(&rpcConfig{}, options.WithOpOnChangeFn(func(cfg interface{}) {
			lock.Lock()
			defer lock.Unlock()
			// 关闭所有旧连接池
			for name, pool := range pools {
				if pool != nil {
					go pool.Close() // 异步关闭，避免阻塞配置更新
					xlog.Infof(context.Background(), "Closed old pool for service: %s", name)
				}
			}

			pools = map[string]*GrpcClientPool{}
		}))
	})

	cfg := config.Get(&rpcConfig{}).(*rpcConfig)
	if err == nil && (cfg.Rpc == nil || cfg.Rpc.Cfgs == nil || len(cfg.Rpc.Cfgs) == 0) {
		err = fmt.Errorf("not configed")
	}
	if err != nil {
		err = fmt.Errorf("rpcx load cfgs error, %s", err)
		return
	}
	r = cfg.Rpc.Cfgs
	return
}
