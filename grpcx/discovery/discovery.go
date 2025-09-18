package discovery

import (
	"os"
	"path/filepath"
	"sync"
)

// KVPair 服务实例键值对（rpcx 风格）
type KVPair struct {
	Key   string
	Value string
	// 可扩展：Metadata map[string]string, Weight int, Healthy bool 等
}

// ServiceDiscoveryFilter 自定义过滤器
type ServiceDiscoveryFilter func(kvp *KVPair) bool

// ServiceDiscovery 服务发现接口（参考 rpcx）
type ServiceDiscovery interface {
	GetServices() []*KVPair
	WatchService() chan []*KVPair // 无 context，适合简单场景
	RemoveWatcher(ch chan []*KVPair)
	Clone(servicePath string) (ServiceDiscovery, error)
	SetFilter(ServiceDiscoveryFilter)
	Close()
}

// CachedServiceDiscovery 带缓存的服务发现（你已实现，稍作优化）
type CachedServiceDiscovery struct {
	threshold  int
	cachedFile string
	cached     []*KVPair

	chansLock sync.RWMutex
	chans     map[chan []*KVPair]chan []*KVPair

	ServiceDiscovery
}

// CacheDiscovery 包装器，支持缓存降级
func CacheDiscovery(threshold int, cachedFile string, discovery ServiceDiscovery) ServiceDiscovery {
	if cachedFile == "" {
		cachedFile = ".cache/discovery.json"
	}

	cachedFileDir := filepath.Dir(cachedFile)
	if _, err := os.Stat(cachedFileDir); os.IsNotExist(err) {
		_ = os.MkdirAll(cachedFileDir, 0755)
	}

	return &CachedServiceDiscovery{
		threshold:        threshold,
		cachedFile:       cachedFile,
		ServiceDiscovery: discovery,
		chans:            make(map[chan []*KVPair]chan []*KVPair),
	}
}
