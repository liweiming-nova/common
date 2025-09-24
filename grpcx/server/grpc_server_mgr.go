package server

import (
	"fmt"
	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/utils"
	"google.golang.org/grpc"
	"sync"
	"time"
)

var (
	once    sync.Once
	lock    sync.RWMutex
	servers map[string]*GrpcServer
)

func init() {
	servers = map[string]*GrpcServer{}
}

type rpcConfig struct {
	Rpc *struct {
		Cfgs map[string]*GrpcConfig `toml:"server"`
	} `toml:"rpc"`
}

type GrpcConfig struct {
	DialAddr         string        `toml:"addr"`
	DialReadTimeout  time.Duration `toml:"read_timeout"`
	DialWriteTimeout time.Duration `toml:"write_timeout"`

	// register
	Register string `toml:"register"`
	// zookeeper register
	RegisterZkServers        []string      `toml:"register_zk_servers"`
	RegisterZkBasePath       string        `toml:"register_zk_basepath"`
	RegisterZkUpdateInterval time.Duration `toml:"register_zk_update_interval"`

	// nacos register
	RegisterNcServers     []string `toml:"register_nc_servers"`
	RegisterNcNamespaceId string   `toml:"register_nc_namespace_id"`
	RegisterNcCacheDir    string   `toml:"register_nc_cache_dir"`
	RegisterNcLogDir      string   `toml:"register_nc_log_dir"`
	RegisterNcLogLevel    string   `toml:"register_nc_log_level"`
	RegisterNcAccessKey   string   `toml:"register_nc_access_key"`
	RegisterNcSecretKey   string   `toml:"register_nc_secret_key"`

	// etcd register
	EtcdEndpoints   []string      `toml:"etcd_endpoints"`
	EtcdDialTimeout time.Duration `toml:"etcd_dial_timeout"`
	EtcdLeaseTTL    int64         `toml:"etcd_lease_ttl"`
	ServiceName     string        `toml:"service_name"`

	// log plugin
	EnableLogPlugin   bool `toml:"enable_log_plugin"`
	EnableRequestLog  bool `toml:"enable_request_log"`
	EnableResponseLog bool `toml:"enable_response_log"`
	EnableErrorLog    bool `toml:"enable_error_log"`
	LogRequestArgs    bool `toml:"log_request_args"`
	LogResponseResult bool `toml:"log_response_result"`
	LogMaxLength      int  `toml:"log_max_length"`

	// serialization
	SerializeType string `toml:"serialize_type"` // 序列化类型: json, protobuf, msgpack, thrift

}

func loadCfg(name string) (r *GrpcConfig, err error) {
	var cfgs map[string]*GrpcConfig
	if cfgs, err = loadCfgs(); err != nil {
		return
	}
	if r = cfgs[name]; r == nil {
		err = fmt.Errorf("rpcx#%s not configed", name)
		return
	}
	return
}

func StartServer(name string, registerFunc func(*grpc.Server), interceptors []grpc.UnaryServerInterceptor) (err error) {
	lock.Lock()
	defer lock.Unlock()
	cfg, err := loadCfg(name)
	if err != nil {
		return err
	}

	server, err := NewGrpcServer(cfg, name, registerFunc, interceptors...)
	if err != nil {
		return err
	}

	servers[name] = server

	utils.Lock(1)

	return server.Start()
}

func StopServer(name string) (err error) {
	defer utils.Unlock()
	lock.Lock()
	server, ok := servers[name]
	if !ok {
		return fmt.Errorf("server %s not found", name)
	}

	return server.Stop()
}

func loadCfgs() (r map[string]*GrpcConfig, err error) {
	r = map[string]*GrpcConfig{}

	cfg := config.Get(&rpcConfig{}).(*rpcConfig)
	if err == nil && (cfg.Rpc == nil || cfg.Rpc.Cfgs == nil || len(cfg.Rpc.Cfgs) == 0) {
		err = fmt.Errorf("not configed")
	}
	if err != nil {
		err = fmt.Errorf("rpc load cfgs error, %s", err)
		return
	}
	r = cfg.Rpc.Cfgs
	return
}
