package register

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liweiming-nova/common/etcd"
	"github.com/liweiming-nova/common/grpcx/instance"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdRegister struct {
	serviceKey string // 在 etcd 中的注册路径，如 /services/user-service/127.0.0.1:8080
	leaseID    clientv3.LeaseID
	cancel     context.CancelFunc
	client     *clientv3.Client
}

// NewEtcdRegister 创建新实例，需后续调用 SetClient 初始化 client
func NewEtcdRegister() *EtcdRegister {
	return &EtcdRegister{
		client: etcd.Get().DefaultClient(),
	}
}

// Register 将服务注册到 etcd
// serviceName: 服务名，如 "user-service"
// address: 服务地址，如 "127.0.0.1:8080"
// metadata: 可选元数据，如 {"version": "v1.2", "env": "prod"}
func (r *EtcdRegister) Register(serviceName string, address string, metadata map[string]string) error {

	if serviceName == "" || address == "" {
		return fmt.Errorf("service name and address cannot be empty")
	}

	// 构造 key：/services/user-service/127.0.0.1:8080
	serviceKey := fmt.Sprintf("/services/%s/%s", strings.TrimPrefix(serviceName, "/"), strings.TrimPrefix(address, "/"))
	r.serviceKey = serviceKey

	// 创建租约（默认 15 秒，自动续期）
	lease := clientv3.NewLease(r.client)
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	// 申请租约（TTL 单位：秒）
	leaseResp, err := lease.Grant(ctx, 15)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to grant lease: %w", err)
	}
	r.leaseID = leaseResp.ID

	// 构造 value：统一使用公共实例编码
	valueBytes, err := instance.Encode(&instance.ServiceInstance{
		Address:      address,
		Metadata:     metadata,
		RegisteredAt: time.Now().Unix(),
	})
	if err != nil {
		lease.Revoke(ctx, r.leaseID)
		cancel()
		return fmt.Errorf("failed to marshal service metadata: %w", err)
	}

	// 使用租约创建键值对（自动续约）
	putCtx, putCancel := context.WithTimeout(ctx, 5*time.Second)
	defer putCancel()

	_, err = r.client.Put(putCtx, serviceKey, string(valueBytes), clientv3.WithLease(r.leaseID))
	if err != nil {
		lease.Revoke(ctx, r.leaseID)
		cancel()
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 启动租约自动续期协程（关键！）
	go func() {
		kaChan, err := lease.KeepAlive(context.Background(), r.leaseID)
		if err != nil {
			fmt.Printf("[EtcdRegister] KeepAlive failed: %v\n", err)
			return
		}

		for {
			select {
			case _, ok := <-kaChan:
				if !ok {
					fmt.Println("[EtcdRegister] KeepAlive channel closed")
					return
				}
				// 正常收到 keep-alive 响应
			case <-ctx.Done():
				lease.Close() // 可选：关闭 lease
				return
			}
		}
	}()

	fmt.Printf("[EtcdRegister] Service registered: %s -> %s\n", serviceKey, string(valueBytes))
	return nil
}

// Unregister 从 etcd 注销服务
func (r *EtcdRegister) Unregister(serviceName string, address string) error {
	if r.client == nil {
		return fmt.Errorf("etcd client not initialized")
	}

	// 如果没有注册过，直接返回
	if r.serviceKey == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 先撤销租约（会自动删除所有关联的 key）
	if r.leaseID != 0 {
		_, err := r.client.Lease.Revoke(ctx, r.leaseID)
		if err != nil {
			return fmt.Errorf("failed to revoke lease: %w", err)
		}
	}

	// 清理状态
	r.serviceKey = ""
	r.leaseID = 0

	// 取消续约协程
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}

	fmt.Printf("[EtcdRegister] Service unregistered: %s\n", r.serviceKey)
	return nil
}

// GetServiceKey 获取当前注册的服务 key（用于调试）
func (r *EtcdRegister) GetServiceKey() string {
	return r.serviceKey
}

// Close 关闭资源（释放客户端？一般由外部管理）
func (r *EtcdRegister) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}
