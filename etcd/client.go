package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/liweiming-nova/common/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"io/ioutil"
	"sync"
	"time"

	"go.uber.org/zap" // 推荐日志库，可替换为 log.Println
)

type Config struct {
	ETCD *struct {
		Endpoints   []string      `toml:"endpoints" yaml:"endpoints" mapstructure:"endpoints"`
		DialTimeout time.Duration `toml:"dial_timeout" yaml:"dial_timeout" mapstructure:"dial_timeout"`
		Username    string        `toml:"username" yaml:"username" mapstructure:"username"`
		Password    string        `toml:"password" yaml:"password" mapstructure:"password"`
		LeaseTTL    int64         `toml:"lease_ttl" yaml:"lease_ttl" mapstructure:"lease_ttl"`
		Prefix      string        `toml:"prefix" yaml:"prefix" mapstructure:"prefix"`

		TLS struct {
			CertFile string `toml:"cert_file" yaml:"cert_file" mapstructure:"cert_file"`
			KeyFile  string `toml:"key_file" yaml:"key_file" mapstructure:"key_file"`
			CAFile   string `toml:"ca_file" yaml:"ca_file" mapstructure:"ca_file"`
		} `toml:"tls" yaml:"tls" mapstructure:"tls"`
	} `toml:"etcd" yaml:"etcd" mapstructure:"etcd"`
}

var defaultClient *Client
var once sync.Once

type Client struct {
	client *clientv3.Client
	cfg    *Config
}

// Start 初始化 etcd client，支持 TLS 和认证
func (c *Client) Start() error {
	cfg := config.Get(&Config{}).(*Config) // 假设 config.Get 是你的配置获取函数
	c.cfg = cfg

	if cfg.ETCD == nil {
		return fmt.Errorf("etcd configuration is not provided")
	}

	// 设置默认值
	if len(cfg.ETCD.Endpoints) == 0 {
		return fmt.Errorf("etcd endpoints cannot be empty")
	}
	if cfg.ETCD.DialTimeout <= 0 {
		cfg.ETCD.DialTimeout = 5 * time.Second // 默认 5 秒
	}
	if cfg.ETCD.LeaseTTL <= 0 {
		cfg.ETCD.LeaseTTL = 15 // 默认 15 秒（介于 10~30 之间）
	}

	// 构建 clientv3 配置
	cliCfg := clientv3.Config{
		Endpoints:   cfg.ETCD.Endpoints,
		DialTimeout: cfg.ETCD.DialTimeout,
		Username:    cfg.ETCD.Username,
		Password:    cfg.ETCD.Password,
	}

	// 如果配置了 TLS，则启用安全连接
	if cfg.ETCD.TLS.CertFile != "" || cfg.ETCD.TLS.KeyFile != "" || cfg.ETCD.TLS.CAFile != "" {
		tlsConfig, err := newTLSConfig(struct{ CertFile, KeyFile, CAFile string }(cfg.ETCD.TLS))
		if err != nil {
			return fmt.Errorf("failed to create TLS config: %w", err)
		}
		cliCfg.TLS = tlsConfig
	}

	// 创建客户端
	client, err := clientv3.New(cliCfg)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}

	// 可选：测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Status(ctx, cfg.ETCD.Endpoints[0]); err != nil {
		client.Close()
		return fmt.Errorf("etcd connection failed: %w", err)
	}

	c.client = client
	zap.L().Info("etcd client started successfully", zap.Strings("endpoints", cfg.ETCD.Endpoints))
	return nil
}

// newTLSConfig 根据配置构建 TLS 配置
func newTLSConfig(tlsCfg struct {
	CertFile, KeyFile, CAFile string
}) (*tls.Config, error) {
	if tlsCfg.CertFile == "" || tlsCfg.KeyFile == "" || tlsCfg.CAFile == "" {
		return nil, fmt.Errorf("TLS requires cert_file, key_file, and ca_file to be set together")
	}

	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(tlsCfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// GetDefaultClient 获取单例客户端
func Get() *Client {
	once.Do(func() {
		defaultClient = &Client{}
		if err := defaultClient.Start(); err != nil {
			panic(fmt.Sprintf("failed to start etcd client: %v", err))
		}
	})
	return defaultClient
}

func (c *Client) DefaultClient() *clientv3.Client {
	return c.client
}

// Close 关闭客户端（可选）
func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
