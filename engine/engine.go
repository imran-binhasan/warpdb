package engine

import "time"

type StorageEngine interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Del(key string) error
	Exists(key string) bool
	Incr(key string) (int, error)
	Decr(key string) (int, error)
	Expire(key string, ttl time.Duration) error
	TTL(key string) (time.Duration, error)
	Persist(key string) error
	Keys(pattern string) []string
	FlushAll()
	Size() int

	LPush(key string, values ...string) int
	RPush(key string, values ...string) int
	LPop(key string) (string, error)
	RPop(key string) (string, error)
	LLen(key string) int
	LRange(key string, start, stop int) []string
	LIndex(key string, index int) (string, error)

	SAdd(key string, members ...string) int
	SRem(key string, members ...string) int
	SMembers(key string) []string
	SIsMember(key, member string) bool
	SCard(key string) int
	SPop(key string) (string, error)
	SRandMember(key string) (string, error)

	HSet(key, field, value string) int
	HGet(key, field string) (string, error)
	HDel(key string, fields ...string) int
	HGetAll(key string) []string
	HKeys(key string) []string
	HVals(key string) []string
	HExists(key, field string) bool
	HLen(key string) int
	HMSet(key string, fields map[string]string)
	HMGet(key string, fields ...string) []string
}
