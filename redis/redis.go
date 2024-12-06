package myredis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// CreateRedisInstance 根据操作类型对Redis执行操作。
func CreateRedisInstance(opType, key string, values ...string) (interface{}, error) {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "123456",
	})

	// 测试连接
	ctx := context.Background()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("无法连接到Redis: %w", err)
	}
	fmt.Println("成功连接到Redis: ", pong)

	// 根据操作类型执行相应操作
	switch opType {
	case "get":
		// 检查键是否存在
		exists, err := rdb.Exists(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("检查键存在失败: %w", err)
		}
		return exists > 0, nil
	case "set":
		if len(values) == 0 {
			return nil, fmt.Errorf("缺少值参数")
		}
		err := rdb.Set(ctx, key, values[0], 0).Err()
		if err != nil {
			return nil, fmt.Errorf("设置值失败: %w", err)
		}
		return "", nil
	default:
		return nil, fmt.Errorf("不支持的操作类型: %s", opType)
	}
}
