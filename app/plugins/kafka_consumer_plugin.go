package plugins

import (
	"context"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/liweiming-nova/common/config"
	"github.com/liweiming-nova/common/xlog"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type HandlerFunc func(msg *ckafka.Message) error

type KafkaConsumerCfg struct {
	Brokers        []string `toml:"brokers"`
	Topics         []string `toml:"topics"`
	GroupID        string   `toml:"group_id"`
	MaxRetries     int      `toml:"max_retries"`
	WorkerPoolSize int      `toml:"worker_pool_size"`
	FetchMaxBytes  int      `toml:"fetch_max_bytes"`
	MaxPollRecords int      `toml:"max_poll_records"` // 每次拉取的最大消息数
}

// KafkaCfg Kafka配置
type KafkaCfg struct {
	Kafka *struct {
		Consumer *KafkaConsumerCfg `toml:"consumer"`
	} `toml:"kafka"`
}

type KafkaConsumerPlugin struct {
	ctx        context.Context
	getHandler func() HandlerFunc
	handler    HandlerFunc
	cfg        *KafkaConsumerCfg
	consumer   *ckafka.Consumer
	antsPool   *ants.Pool
}

func NewKafkaConsumerPlugin(getHandler func() HandlerFunc) *KafkaConsumerPlugin {
	return &KafkaConsumerPlugin{
		ctx:        context.Background(),
		getHandler: getHandler,
	}
}

func (p *KafkaConsumerPlugin) Start(ctx *PluginContext) error {
	p.loadCfg()
	p.handler = p.getHandler()
	if p.handler == nil {
		panic("handler is nil")
	}
	antsPoolSize := p.cfg.WorkerPoolSize
	if antsPoolSize <= 0 {
		// 默认核心数
		antsPoolSize = runtime.NumCPU()
	}

	antsPool, err := ants.NewPool(antsPoolSize)
	if err != nil {
		panic(err)
	}
	p.antsPool = antsPool

	conf := &ckafka.ConfigMap{
		"bootstrap.servers":  strings.Join(p.cfg.Brokers, ","),
		"group.id":           p.cfg.GroupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	}

	if p.cfg.FetchMaxBytes > 0 {
		_ = conf.SetKey("fetch.max.bytes", p.cfg.FetchMaxBytes)
	}
	if p.cfg.MaxPollRecords > 0 {
		_ = conf.SetKey("max.poll.records", p.cfg.MaxPollRecords)
	}

	// 创建消费者实例
	consumer, err := ckafka.NewConsumer(conf)
	if err != nil {
		return err
	}
	p.consumer = consumer

	// 订阅主题
	if err := consumer.SubscribeTopics(p.cfg.Topics, nil); err != nil {
		_ = consumer.Close()
		return err
	}

	go p.pollMessage()

	log.Println("Started Kafka consumer successfully")
	return nil
}

func (p *KafkaConsumerPlugin) pollMessage() {
	for {
		msg, err := p.consumer.ReadMessage(100 * time.Millisecond)
		if err != nil {
			if kerr, ok := err.(ckafka.Error); ok {
				if kerr.Code() == ckafka.ErrTimedOut {
					continue
				}
				if kerr.Code() == ckafka.ErrUnknownTopicOrPart {
					xlog.Errorf(p.ctx, "Kafka topic error:%v", err)
					continue
				}
				if kerr.Code() == ckafka.ErrDestroy {
					log.Println("Consumer is closing, exiting polling loop")
					return
				}
			}
		}

		_ = p.antsPool.Submit(func() {
			p.processMessage(msg)
		})
	}
}

func (p *KafkaConsumerPlugin) processMessage(msg *ckafka.Message) {
	var lastErr error
	maxRetries := p.cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	for i := 0; i < int(maxRetries+1); i++ {
		if i > 0 {
			time.Sleep(time.Second)
		}

		err := p.handler(msg)
		if err == nil {
			if _, err := p.consumer.CommitMessage(msg); err != nil {
				xlog.Errorf(p.ctx, "Failed to commit message:%v", zap.Error(err))
			}
			return
		}

		lastErr = err
	}

	xlog.Errorf(p.ctx,
		"All retry attempts failed for message: topic=%s, partition=%d, offset=%d, err=%v",
		*msg.TopicPartition.Topic,
		msg.TopicPartition.Partition,
		int64(msg.TopicPartition.Offset),
		lastErr,
	)
}

func (p *KafkaConsumerPlugin) Stop() error {
	p.consumer.Close()

	p.antsPool.Release()

	log.Println("Stopped Kafka consumer successfully")
	return nil
}
func (p *KafkaConsumerPlugin) BeforeStart(ctx *PluginContext) error {
	return nil
}

func (p *KafkaConsumerPlugin) loadCfg() {
	if os.Getenv("PLUGIN_TEST") == "true" {
		p.cfg = &KafkaConsumerCfg{
			Brokers: []string{"192.168.21.23:9092"},
			Topics:  []string{"test"},
			GroupID: "fund_common_test",
		}
		return
	}
	cfg := config.Get(&KafkaCfg{}).(*KafkaCfg)
	if cfg.Kafka == nil {
		panic("kafka config is nil")
	}

	p.cfg = cfg.Kafka.Consumer
}
