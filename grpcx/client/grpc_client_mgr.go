package client

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/config/options"
	"github.com/liweiming-nova/common/grpcx/discovery"
	"github.com/liweiming-nova/common/grpcx/instance"
	"github.com/liweiming-nova/common/utils"
	"github.com/liweiming-nova/common/xlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const (
	// 失败重试机制
	FailModeNothing  = "nothing"  // 不重试
	FailModeLocal    = "local"    // 原地重试（同一连接）
	FailModeFailover = "failover" // 降级重试（切换连接）

	// 节点选择机制
	SelectModeRoundRobin = "round_robin" // 轮询
	SelectModeRandom     = "random"      // 随机
	SelectModeScore      = "score"       // 评分
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
	// retry
	RetryTimes int `toml:"retry_times"` // 重试次数（针对 local/failover）
}

// GrpcClientPool 是一个固定大小的 gRPC 客户端连接池
// 使用 round-robin 算法选择客户端
type GrpcClientPool struct {
	count   uint64
	index   uint64
	clients []*grpc.ClientConn
	mu      sync.RWMutex

	clientTimeout time.Duration
	failMode      string
	selectMode    string
	retryTimes    int

	// 服务发现
	discovery   discovery.ServiceDiscovery
	serviceName string
	addrIdx     uint64

	// 选择器
	selector Selector

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

	if cfg.DialFailMode == "" {
		cfg.DialFailMode = FailModeNothing
	}
	if cfg.DialSelectMode == "" {
		cfg.DialSelectMode = SelectModeRoundRobin
	}
	if cfg.RetryTimes < 0 {
		cfg.RetryTimes = 0
	}

	pool := &GrpcClientPool{
		count:         uint64(count),
		clients:       make([]*grpc.ClientConn, count),
		failMode:      cfg.DialFailMode,
		selectMode:    cfg.DialSelectMode,
		clientTimeout: cfg.DialTimeout,
		discovery:     discovery,
		serviceName:   cfg.ServiceName,
		retryTimes:    cfg.RetryTimes,
	}

	// 初始化选择器（默认轮询）
	pool.selector = GetSelector(cfg.DialSelectMode, pool)

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
	if p.discovery == nil {
		return nil, errors.New("discovery is nil")
	}

	kvPairs := p.discovery.GetServices()
	if len(kvPairs) == 0 {
		return nil, errors.New("discovery has no services")
	}
	// 根据选择策略挑选目标（优先使用自定义选择器）
	target := ""
	if p.selector != nil {
		target = p.selector.Pick(kvPairs)
	}
	if target == "" { // 兜底使用内置逻辑
		target = p.pickTarget(kvPairs)
	}
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

// pickTarget 根据 selectMode 选择目标地址
func (p *GrpcClientPool) pickTarget(kvPairs []*discovery.KVPair) string {
	n := len(kvPairs)
	if n == 0 {
		return ""
	}
	mode := strings.ToLower(strings.TrimSpace(p.selectMode))
	switch mode {
	case SelectModeRandom:
		idx := rand.Intn(n)
		return instance.ExtractAddress(kvPairs[idx].Value)
	case SelectModeScore:
		// 简单评分选择：抽样 3 个，选分最高；若不足 3 个则全量找最高
		bestIdx, bestScore := 0, float64(0)
		sample := 3
		if n < sample {
			sample = n
		}
		for i := 0; i < sample; i++ {
			idx := i
			if n > sample {
				idx = rand.Intn(n)
			}
			score := instance.ExtractScore(kvPairs[idx].Value)
			if score > bestScore {
				bestScore = score
				bestIdx = idx
			}
		}
		return instance.ExtractAddress(kvPairs[bestIdx].Value)
	case SelectModeRoundRobin:
		fallthrough
	default:
		idx := int(atomic.AddUint64(&p.addrIdx, 1) % uint64(n))
		return instance.ExtractAddress(kvPairs[idx].Value)
	}
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

	// 注入 trace_id 到 outgoing metadata
	ctx = withTraceID(ctx)

	// 重试策略
	switch strings.ToLower(strings.TrimSpace(p.failMode)) {
	case FailModeNothing:
		return client.Invoke(ctx, normalized, req, resp)
	case FailModeLocal: // 原地重试：同一连接重试
		var lastErr error
		for i := 0; i <= p.retryTimes; i++ {
			if err := client.Invoke(ctx, normalized, req, resp); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
		return lastErr
	case FailModeFailover: // 降级重试：更换节点
		var lastErr error
		for i := 0; i <= p.retryTimes; i++ {
			cc := p.Get()
			if cc == nil {
				lastErr = fmt.Errorf("no available client in pool")
				continue
			}
			if err := cc.Invoke(ctx, normalized, req, resp); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
		return lastErr
	default:
		return client.Invoke(ctx, normalized, req, resp)
	}
}

// withTraceID 确保在 outgoing metadata 中携带 trace_id
func withTraceID(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	// 优先从已有的 outgoing/incoming metadata 读取
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if vals := md.Get(xlog.TraceId); len(vals) > 0 && vals[0] != "" {
			return ctx
		}
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(xlog.TraceId); len(vals) > 0 && vals[0] != "" {
			return metadata.AppendToOutgoingContext(ctx, xlog.TraceId, vals[0])
		}
	}
	// 兜底：生成一个新的 trace_id（无连贯上游时）
	traceID := strings.ReplaceAll(utils.UUID(), "-", "")
	return metadata.AppendToOutgoingContext(ctx, xlog.TraceId, traceID)
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
