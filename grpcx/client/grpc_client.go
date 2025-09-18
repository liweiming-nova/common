package client

import (
	"github.com/liweiming-nova/common/grpcx/discovery"
)

func NewRpcClientPool(cfg *Cfg) (r *GrpcClientPool, err error) {
	maxActive := cfg.PoolMaxActive
	dis, err := buildDialDiscovery(cfg)
	if err != nil {
		return
	}
	r, err = NewGrpcClientPool(maxActive, cfg, dis)
	return
}

func buildDialDiscovery(cfg *Cfg) (r discovery.ServiceDiscovery, err error) {
	switch cfg.DialDiscovery {
	case "multiple_servers_discovery":
	case "zookeeper_discovery":
	case "nacos_discovery":
	case "etcd_discovery":
		r, err = discovery.NewEtcdDiscovery(cfg.DialAddrs, "/services/"+cfg.ServiceName, cfg.EtcdDialTimeout)
	}
	return
}
