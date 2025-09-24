package client

import (
	"github.com/liweiming-nova/common/grpcx/discovery"
)

func NewRpcClientPool(name string, cfg *Cfg) (r *GrpcClientPool, err error) {
	maxActive := cfg.PoolMaxActive
	dis, err := buildDialDiscovery(name, cfg)
	if err != nil {
		return
	}
	r, err = NewGrpcClientPool(maxActive, cfg, dis)
	return
}

func buildDialDiscovery(name string, cfg *Cfg) (r discovery.ServiceDiscovery, err error) {
	r, err = discovery.NewEtcdDiscovery("/services/" + name)
	return
}
