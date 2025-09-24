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
	r, err = discovery.NewEtcdDiscovery("/services/" + cfg.ServiceName)
	return
}
