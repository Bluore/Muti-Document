package memory

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/redis/go-redis/v9"
)

// 默认键前缀与过期时间
const (
	DefaultKeyPrefix = "agent:checkpoint:"
	DefaultTTL       = 24 * time.Hour
)

// ErrNilClient redis 客户端未初始化
var ErrNilClient = errors.New("redis client is nil")

// RedisStore 基于 Redis 的 eino 检查点存储, 用于缓存会话记忆/运行态
type RedisStore struct {
	cli    *redis.Client
	prefix string
	ttl    time.Duration
}

// 分别实现 eino 的读写与删除接口
var (
	_ adk.CheckPointStore   = (*RedisStore)(nil)
	_ adk.CheckPointDeleter = (*RedisStore)(nil)
)

// RedisOption 自定义 RedisStore
type RedisOption func(*RedisStore)

// WithKeyPrefix 设置键前缀
func WithKeyPrefix(prefix string) RedisOption {
	return func(s *RedisStore) {
		if prefix != "" {
			s.prefix = prefix
		}
	}
}

// WithTTL 设置键过期时间, 传 0 表示永不过期
func WithTTL(ttl time.Duration) RedisOption {
	return func(s *RedisStore) {
		s.ttl = ttl
	}
}

// NewRedisStore 创建 Redis 检查点存储
func NewRedisStore(cli *redis.Client, opts ...RedisOption) *RedisStore {
	s := &RedisStore{
		cli:    cli,
		prefix: DefaultKeyPrefix,
		ttl:    DefaultTTL,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Set 写入检查点, 按配置的 TTL 过期
func (s *RedisStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	if s.cli == nil {
		return ErrNilClient
	}
	return s.cli.Set(ctx, s.key(checkPointID), checkPoint, s.ttl).Err()
}

// Get 读取检查点, 不存在时 found 为 false
func (s *RedisStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	if s.cli == nil {
		return nil, false, ErrNilClient
	}
	b, err := s.cli.Get(ctx, s.key(checkPointID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Delete 删除检查点
func (s *RedisStore) Delete(ctx context.Context, checkPointID string) error {
	if s.cli == nil {
		return ErrNilClient
	}
	return s.cli.Del(ctx, s.key(checkPointID)).Err()
}

// key 拼接实际存储键
func (s *RedisStore) key(checkPointID string) string {
	return s.prefix + checkPointID
}
