package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/liweiming-nova/common/xlog"
	"time"
)

type RedisLocker struct {
	client *redis.Client
}

func NewRedisLocker(c *redis.Client) *RedisLocker {
	return &RedisLocker{client: c}
}

const (
	releaseLockScript = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
		    redis.call("DEL", KEYS[1])
			redis.call("PUBLISH", ARGV[2], "ok")
			return 1
		else
			return 0
		end
	`
)

func (d *RedisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (string, error) {
	token := genLockToken()
	channel := genChannelKey(key)

	if acquired := d.tryAcquire(ctx, key, token, ttl); acquired {
		return token, nil
	}

	subscriber := d.client.Subscribe(ctx, channel)
	defer subscriber.Close()

	if _, err := subscriber.Receive(ctx); err != nil {
		if acquired := d.tryAcquire(ctx, key, token, ttl); acquired {
			return token, nil
		}
		return "", err
	}

	if acquired := d.tryAcquire(ctx, key, token, ttl); acquired {
		return token, nil
	}

	for {
		select {
		case <-ctx.Done():
			xlog.Errorf(ctx, "lock acquire timeout key:%s", key)
			return "", ErrLockTimeout

		case msg, ok := <-subscriber.Channel():
			if !ok {
				if acquired := d.tryAcquire(ctx, key, token, ttl); acquired {
					return token, nil
				}
				return "", ErrAlreadyLocked
			}

			if msg.Channel == channel {
				if acquired := d.tryAcquire(ctx, key, token, ttl); acquired {
					return token, nil
				}
			}
		}
	}
}

func (d *RedisLocker) tryAcquire(ctx context.Context, key, token string, ttl time.Duration) bool {
	acquired, err := d.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		xlog.Errorf(ctx, "try acquire failed key:%s", key)
		return false
	}
	return acquired
}

func (d *RedisLocker) Release(key, token string) (bool, error) {
	channel := genChannelKey(key)

	result, err := d.client.Eval(context.Background(), releaseLockScript, []string{key}, token, channel).Result()
	if err != nil {
		return false, ErrLockNetwork
	}

	success := result.(int64) == 1
	if !success {
		// 可选：区分“token 不匹配”错误
		return false, ErrReleaseFailed
	}

	return true, nil
}

// genLockToken 生成唯一的锁值
func genLockToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func genChannelKey(key string) string {
	return fmt.Sprintf("fund_hub:lock:%s:released", key)
}
