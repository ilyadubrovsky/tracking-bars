package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ilyadubrovsky/bars"
	"github.com/redis/go-redis/v9"
	"grades-service/pkg/logging"
	"time"
)

const (
	expiration = 3 * time.Minute
)

type gradesCache struct {
	client client
	logger *logging.Logger
}

func NewGradesRedis(client client, logger *logging.Logger) *gradesCache {
	return &gradesCache{client: client, logger: logger}
}

func (c *gradesCache) Set(ctx context.Context, key string, pt *bars.ProgressTable) error {
	c.logger.Tracef("Redis: SET for key: %s", key)
	ptJSON, err := json.Marshal(pt)
	if err != nil {
		return fmt.Errorf("json marshal: %v", err)
	}

	if err = c.client.Set(ctx, key, ptJSON, expiration).Err(); err != nil {
		return fmt.Errorf("redis set: %v", err)
	}

	return nil
}

func (c *gradesCache) Get(ctx context.Context, key string) (*bars.ProgressTable, error) {
	c.logger.Tracef("Redis: GET for key: %s", key)

	ptJSON, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("redis get: %v", err)
	}

	var pt bars.ProgressTable
	if err = json.Unmarshal([]byte(ptJSON), &pt); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	return &pt, nil
}
