package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pilu/go-base62"
	"time"
)
import "github.com/go-redis/redis"

// redis key
const (
	// global counter
	URLIdKey = "next.url.id"
	// mapping the shortlink to the url
	ShortlinkKey = "short:%s:url"
	// mapping the hash of the url to the shortlink
	URLHashKey = "urlhash:%s:url"
	// mapping the shortlink to the detail of url
	ShortlinkDetailKey = "shortlink:%s:detail"
)

// redis client
type RedisClient struct {
	Client *redis.Client
}

// the detail of the shorklink
type ShortlinkDetail struct {
	URL                 string        `json:"url"`
	CreatedAt           string        `json:"created_at"`
	ExpirationInMinutes time.Duration `json:"expiration_in_minutes'"`
}

// create a redis client
func NewRedisClient(addr string, password string, db int) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if _, err := client.Ping().Result(); err != nil {
		panic(err)
	}

	return &RedisClient{Client: client}
}

// get shortlink by url
func (r *RedisClient) Shorten(url string, exp int64) (string, error) {
	// convert url to sha1 hash
	h := toSha1(url)

	// fetch it if the url is cached
	d, err := r.Client.Get(fmt.Sprintf(URLHashKey, h)).Result()
	if err == redis.Nil {
		// not existed, nothing to do
	} else if err != nil {
		return "", err
	} else {
		if d == "{}" {
			// expiration, nothing to do
		} else {
			return d, nil
		}
	}

	// increase the global counter
	err = r.Client.Incr(URLIdKey).Err()
	if err != nil {
		return "", err
	}

	// encode global counter to base62
	id, err := r.Client.Get(URLIdKey).Int()
	if err != nil {
		return "", err
	}
	eid := base62.Encode(id)

	// store redis: shortlink -> origin url
	err = r.Client.Set(fmt.Sprintf(ShortlinkKey, eid), url,
		time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	// store redis: urlHash -> shortlink
	err = r.Client.Set(fmt.Sprintf(URLHashKey, h), eid,
		time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	// create urlDetail
	detail, err := json.Marshal(&ShortlinkDetail{
		URL:                 url,
		CreatedAt:           time.Now().String(),
		ExpirationInMinutes: time.Duration(exp),
	})
	if err != nil {
		return "", err
	}

	// store redis: shortlink -> shortlinkDetail
	err = r.Client.Set(fmt.Sprintf(ShortlinkDetailKey, eid), detail,
		time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	return eid, nil
}

// get the detail of shortlink
func (r *RedisClient) ShortlinkInfo(eid string) (interface{}, error) {
	detail, err := r.Client.Get(fmt.Sprintf(ShortlinkDetailKey, eid)).Result()
	if err == redis.Nil {
		return "", StatusError{
			Code: 404,
			Err:  errors.New("unknown shortlink"),
		}
	} else if err != nil {
		return "", nil
	}

	return detail, err
}

// get url by shortlink
func (r *RedisClient) UnShorten(eid string) (string, error) {
	url, err := r.Client.Get(fmt.Sprintf(ShortlinkKey, eid)).Result()
	if err == redis.Nil {
		return "", StatusError{
			Code: 404,
			Err:  errors.New("unknown shortlink"),
		}
	} else if err != nil {
		return "", nil
	}

	return url, err
}

func toSha1(url string) string {

}
