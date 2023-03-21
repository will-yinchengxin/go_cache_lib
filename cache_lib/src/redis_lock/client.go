package redis_lock

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	FailToGetLock = errors.New("Fail To Get Lock")
)

type Client struct {
	client redis.Cmdable
}

func NewClient(c redis.Cmdable) *Client {
	return &Client{
		client: c,
	}
}

func (c *Client) Lock(ctx context.Context, key string, val string, expiration time.Duration, retry RetryStrategy, timeout time.Duration) (*Lock, error) {
	// Todo: 可以自行传递，或者通过自定义方法获取
	//val := c.valuer()

	/*
		   type Timer struct {
			 C <-chan Time
			 r runtimeTimer
		   }
			type Ticker struct {
				C <-chan Time // The channel on which the ticks are delivered.
				r runtimeTimer
			}
	*/
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		tCtx, cancelFunc := context.WithTimeout(ctx, timeout)
		res, err := c.client.Eval(tCtx, luaLock, []string{key}, val, expiration.Seconds()).Result()
		cancelFunc()
		// 加锁超时了直接返回错误即可
		if err != nil && err == context.DeadlineExceeded {
			return nil, err
		}
		// 加锁成功
		if res == "OK" {
			return newLock(c.client, key, val, expiration), nil
		}
		// 加锁未超时且加锁失败，那就重试几次
		interval, ok := retry.Next()
		if !ok {
			if err == nil {
				err = fmt.Errorf("锁被人持有")
			} else {
				err = fmt.Errorf("最后一次重试错误: %w", err)
			}
			return nil, fmt.Errorf("重试机会耗尽, %w", err)
		}
		if timer == nil {
			timer = time.NewTimer(interval)
		} else {
			timer.Reset(interval)
		}
		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (c *Client) TryLock(ctx context.Context,
	key string, val any, expiration time.Duration) (*Lock, error) {
	ok, err := c.client.SetNX(ctx, key, val, expiration).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, FailToGetLock
	}
	return newLock(c.client, key, val, expiration), nil
}

/*
获取 uuid 的方法，这里可以自定义，或者说用户自己传入

	type Client struct {
		client redis.Cmdable
		valuer func() string
	}
*/
//func (c *Client) valuer() string {
//	return ""
//}
