package discovery

import (
	"context"
	"github.com/liweiming-nova/common/xlog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"strings"
	"sync"
	"time"
)

type EtcdDiscovery struct {
	client      *clientv3.Client
	servicePath string
	filter      ServiceDiscoveryFilter
	mu          sync.RWMutex
	watchers    map[chan []*KVPair]context.CancelFunc
	watchersMu  sync.RWMutex
}

func NewEtcdDiscovery(endpoints []string, servicePath string, dialTimeout time.Duration) (*EtcdDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, err
	}

	d := &EtcdDiscovery{
		client:      cli,
		servicePath: servicePath,
		watchers:    make(map[chan []*KVPair]context.CancelFunc),
	}

	return d, nil
}

func (d *EtcdDiscovery) GetServices() []*KVPair {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := d.client.Get(ctx, d.servicePath, clientv3.WithPrefix())
	if err != nil {
		xlog.Errorf(context.Background(), "EtcdDiscovery GetServices error: %v", err)
		return nil
	}

	var pairs []*KVPair
	for _, kv := range resp.Kvs {
		value := strings.TrimSpace(string(kv.Value))
		if value == "" {
			continue
		}
		pair := &KVPair{
			Key:   string(kv.Key),
			Value: value,
		}
		if d.filter == nil || d.filter(pair) {
			pairs = append(pairs, pair)
		}
	}

	return pairs
}

func (d *EtcdDiscovery) WatchService() chan []*KVPair {
	ch := make(chan []*KVPair, 10)

	ctx, cancel := context.WithCancel(context.Background())

	d.watchersMu.Lock()
	d.watchers[ch] = cancel
	d.watchersMu.Unlock()

	go d.watch(ctx, ch)

	return ch
}

func (d *EtcdDiscovery) watch(ctx context.Context, ch chan []*KVPair) {
	defer func() {
		d.watchersMu.Lock()
		delete(d.watchers, ch)
		d.watchersMu.Unlock()
		close(ch)
	}()

	watchCh := d.client.Watch(ctx, d.servicePath, clientv3.WithPrefix())

	// 初始推送一次
	if pairs := d.GetServices(); len(pairs) > 0 {
		select {
		case ch <- pairs:
		case <-ctx.Done():
			return
		}
	}

	for {
		select {
		case wresp := <-watchCh:
			if wresp.Err() != nil {
				continue
			}
			if pairs := d.GetServices(); len(pairs) > 0 {
				select {
				case ch <- pairs:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *EtcdDiscovery) RemoveWatcher(ch chan []*KVPair) {
	d.watchersMu.Lock()
	if cancel, ok := d.watchers[ch]; ok {
		cancel()
		delete(d.watchers, ch)
	}
	d.watchersMu.Unlock()
}

func (d *EtcdDiscovery) Clone(servicePath string) (ServiceDiscovery, error) {
	return NewEtcdDiscovery(
		[]string{},
		servicePath,
		5*time.Second,
	)
}

func (d *EtcdDiscovery) SetFilter(filter ServiceDiscoveryFilter) {
	d.filter = filter
}

func (d *EtcdDiscovery) Close() {
	d.watchersMu.Lock()
	for _, cancel := range d.watchers {
		cancel()
	}
	d.watchers = nil
	d.watchersMu.Unlock()

	if d.client != nil {
		d.client.Close()
	}
}
