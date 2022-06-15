//
// https://pkg.go.dev/github.com/go-redis/redis/v8
//

package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	redis "github.com/go-redis/redis/v8"
)

type CachedDatabase struct {
	Client *redis.Client
}

const (
	maxIdle   = 10
	maxActive = 300
)

var (
	ErrNil = errors.New("no matching record found in redis database")
	Ctx    = context.TODO()
)

func NewCachedDatabase(address, password string, db int) (*CachedDatabase, error) {
	if address == "" {
		address = ":6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})
	// client := redis.NewClient(&redis.Options{
	// 	Addr:         address,
	// 	DialTimeout:  10 * time.Second,
	// 	ReadTimeout:  30 * time.Second,
	// 	WriteTimeout: 30 * time.Second,
	// 	PoolSize:     10,
	// 	PoolTimeout:  30 * time.Second,
	// 	Password:     password,
	// 	DB:           db,
	// })
	if err := client.Ping(Ctx).Err(); err != nil {
		return nil, err
	}
	return &CachedDatabase{
		Client: client,
	}, nil
}

//setKey set key in redis
func (db *CachedDatabase) SetKey(key string, value interface{}, expiration time.Duration) error {
	cacheEntry, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = db.Client.Set(Ctx, key, cacheEntry, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

//getKey get key in redis
func (db *CachedDatabase) GetKey(key string, src interface{}, ctx context.Context) (interface{}, error) {
	if ctx == nil {
		ctx = Ctx
	}
	val, err := db.Client.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(val), &src)
	if err != nil {
		return nil, err
	}
	return src, nil
}

// MGet get multiple value
func (db *CachedDatabase) MGet(keys []string, ctx context.Context) ([]interface{}, error) {
	if ctx == nil {
		ctx = Ctx
	}
	return db.Client.MGet(ctx, keys...).Result()
}

func (db *CachedDatabase) GetKeyListByPattern(pattern string, ctx context.Context) []string {
	if ctx == nil {
		ctx = Ctx
	}
	return db.Client.Keys(ctx, pattern).Val()
}

func (db *CachedDatabase) ListByKey(key string, src interface{}, ctx context.Context) ([]interface{}, error) {
	if ctx == nil {
		ctx = Ctx
	}
	var results []interface{}
	var cursor uint64
	var keys []string
	var err error
	for {
		keys, cursor, err = db.Client.Scan(ctx, cursor, key+"*", 0).Result()
		if err == redis.Nil || err != nil {
			return results, err
		}
		for _, val := range keys {
			err = json.Unmarshal([]byte(val), &src)
			if err != nil {
				return results, err
			}
			results = append(results, &src)
		}

		if cursor == 0 { // no more keys
			return results, nil
		}
	}
}

func (db *CachedDatabase) Ping() error {
	// s, err := redis.String(db.Do("PING"))
	err := db.Client.Ping(Ctx).Err()
	if err != nil {
		return err
	}
	// fmt.Printf("PING Response = %s\n", s)
	return nil
}
