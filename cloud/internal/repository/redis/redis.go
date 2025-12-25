package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"cloud-system/internal/config"
	"cloud-system/internal/utils"
	"go.uber.org/zap"
)

// Client Redis客户端
type Client struct {
	client *redis.Client
	cfg    *config.Config
}

// NewClient 创建Redis客户端
func NewClient(cfg *config.Config) (*Client, error) {
	ctx := context.Background()
	
	// 解析超时时间
	dialTimeout, _ := time.ParseDuration(cfg.Database.Redis.DialTimeout)
	readTimeout, _ := time.ParseDuration(cfg.Database.Redis.ReadTimeout)
	writeTimeout, _ := time.ParseDuration(cfg.Database.Redis.WriteTimeout)
	
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Database.Redis.Password,
		DB:           cfg.Database.Redis.DB,
		PoolSize:     cfg.Database.Redis.PoolSize,
		MinIdleConns: cfg.Database.Redis.MinIdleConns,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})
	
	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}
	
	utils.Info("Redis connection established",
		zap.String("addr", cfg.GetRedisAddr()),
		zap.Int("db", cfg.Database.Redis.DB),
	)
	
	return &Client{
		client: rdb,
		cfg:    cfg,
	}, nil
}

// GetClient 获取Redis客户端
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.client != nil {
		err := c.client.Close()
		utils.Info("Redis connection closed")
		return err
	}
	return nil
}

// Ping 检查连接
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Set 设置键值
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// Expire 设置过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL 获取键的剩余生存时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// HSet 设置哈希字段
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希字段值
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// Incr 自增
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Decr 自减
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx)
}

// Stats 获取Redis统计信息
func (c *Client) Stats() map[string]interface{} {
	stats := c.client.PoolStats()
	return map[string]interface{}{
		"hits":         stats.Hits,
		"misses":       stats.Misses,
		"timeouts":     stats.Timeouts,
		"total_conns":  stats.TotalConns,
		"idle_conns":   stats.IdleConns,
		"stale_conns":  stats.StaleConns,
	}
}

