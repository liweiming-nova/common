package client

import (
	"math/rand"
	"strings"
	"sync/atomic"

	"github.com/liweiming-nova/common/grpcx/discovery"
	"github.com/liweiming-nova/common/grpcx/instance"
)

// Selector 节点选择接口
type Selector interface {
	// Pick 在 kvPairs 中选择一个目标地址（host:port），返回空字符串表示不可用
	Pick(kvPairs []*discovery.KVPair) string
}

// SelectorFactory 选择器工厂
type SelectorFactory func(p *GrpcClientPool) Selector

var selectorRegistry = map[string]SelectorFactory{}

// RegisterSelector 注册自定义选择器
func RegisterSelector(name string, factory SelectorFactory) {
	selectorRegistry[strings.ToLower(strings.TrimSpace(name))] = factory
}

// GetSelector 根据名称获取选择器，找不到返回默认轮询
func GetSelector(name string, p *GrpcClientPool) Selector {
	if f, ok := selectorRegistry[strings.ToLower(strings.TrimSpace(name))]; ok {
		return f(p)
	}
	return (&roundRobinFactory{}).New(p)
}

// ---- 内置实现 ----

type roundRobinSelector struct{ p *GrpcClientPool }
type roundRobinFactory struct{}

func (f *roundRobinFactory) New(p *GrpcClientPool) Selector { return &roundRobinSelector{p: p} }

func (s *roundRobinSelector) Pick(kvPairs []*discovery.KVPair) string {
	n := len(kvPairs)
	if n == 0 {
		return ""
	}
	idx := int(atomic.AddUint64(&s.p.addrIdx, 1) % uint64(n))
	return instance.ExtractAddress(kvPairs[idx].Value)
}

type randomSelector struct{}
type randomFactory struct{}

func (f *randomFactory) New(p *GrpcClientPool) Selector { return &randomSelector{} }

func (s *randomSelector) Pick(kvPairs []*discovery.KVPair) string {
	n := len(kvPairs)
	if n == 0 {
		return ""
	}
	idx := rand.Intn(n)
	return instance.ExtractAddress(kvPairs[idx].Value)
}

type scoreSelector struct{}
type scoreFactory struct{}

func (f *scoreFactory) New(p *GrpcClientPool) Selector { return &scoreSelector{} }

func (s *scoreSelector) Pick(kvPairs []*discovery.KVPair) string {
	n := len(kvPairs)
	if n == 0 {
		return ""
	}
	// 抽样 3 选最优
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
}

func init() {
	RegisterSelector(SelectModeRoundRobin, func(p *GrpcClientPool) Selector { return (&roundRobinFactory{}).New(p) })
	RegisterSelector(SelectModeRandom, func(p *GrpcClientPool) Selector { return (&randomFactory{}).New(p) })
	RegisterSelector(SelectModeScore, func(p *GrpcClientPool) Selector { return (&scoreFactory{}).New(p) })
}
