package engine

import (
	"time"

	"github.com/imran-binhasan/warpdb/core/store"
)

type MemoryEngine struct {
	store *store.Store
}

var _ StorageEngine = (*MemoryEngine)(nil)

func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{store: store.NewStore()}
}

func (e *MemoryEngine) Set(key, value string) error {
	e.store.Set(key, value)
	return nil
}

func (e *MemoryEngine) Get(key string) (string, error) {
	return e.store.Get(key)
}

func (e *MemoryEngine) Del(key string) error {
	return e.store.Del(key)
}

func (e *MemoryEngine) Exists(key string) bool {
	return e.store.Exists(key)
}

func (e *MemoryEngine) Incr(key string) (int, error) {
	return e.store.Incr(key)
}

func (e *MemoryEngine) Decr(key string) (int, error) {
	return e.store.Decr(key)
}

func (e *MemoryEngine) Expire(key string, ttl time.Duration) error {
	return e.store.Expire(key, ttl)
}

func (e *MemoryEngine) TTL(key string) (time.Duration, error) {
	return e.store.TTL(key)
}

func (e *MemoryEngine) Persist(key string) error {
	return e.store.Persist(key)
}

func (e *MemoryEngine) Keys(pattern string) []string {
	return e.store.Keys(pattern)
}

func (e *MemoryEngine) FlushAll() {
	e.store.FlushAll()
}

func (e *MemoryEngine) Size() int {
	return e.store.Size()
}

func (e *MemoryEngine) LPush(key string, values ...string) int {
	return e.store.LPush(key, values...)
}

func (e *MemoryEngine) RPush(key string, values ...string) int {
	return e.store.RPush(key, values...)
}

func (e *MemoryEngine) LPop(key string) (string, error) {
	return e.store.LPop(key)
}

func (e *MemoryEngine) RPop(key string) (string, error) {
	return e.store.RPop(key)
}

func (e *MemoryEngine) LLen(key string) int {
	return e.store.LLen(key)
}

func (e *MemoryEngine) LRange(key string, start, stop int) []string {
	return e.store.LRange(key, start, stop)
}

func (e *MemoryEngine) LIndex(key string, index int) (string, error) {
	return e.store.LIndex(key, index)
}

func (e *MemoryEngine) SAdd(key string, members ...string) int {
	return e.store.SAdd(key, members...)
}

func (e *MemoryEngine) SRem(key string, members ...string) int {
	return e.store.SRem(key, members...)
}

func (e *MemoryEngine) SMembers(key string) []string {
	return e.store.SMembers(key)
}

func (e *MemoryEngine) SIsMember(key, member string) bool {
	return e.store.SIsMember(key, member)
}

func (e *MemoryEngine) SCard(key string) int {
	return e.store.SCard(key)
}

func (e *MemoryEngine) SPop(key string) (string, error) {
	return e.store.SPop(key)
}

func (e *MemoryEngine) SRandMember(key string) (string, error) {
	return e.store.SRandMember(key)
}

func (e *MemoryEngine) HSet(key, field, value string) int {
	return e.store.HSet(key, field, value)
}

func (e *MemoryEngine) HGet(key, field string) (string, error) {
	return e.store.HGet(key, field)
}

func (e *MemoryEngine) HDel(key string, fields ...string) int {
	return e.store.HDel(key, fields...)
}

func (e *MemoryEngine) HGetAll(key string) []string {
	return e.store.HGetAll(key)
}

func (e *MemoryEngine) HKeys(key string) []string {
	return e.store.HKeys(key)
}

func (e *MemoryEngine) HVals(key string) []string {
	return e.store.HVals(key)
}

func (e *MemoryEngine) HExists(key, field string) bool {
	return e.store.HExists(key, field)
}

func (e *MemoryEngine) HLen(key string) int {
	return e.store.HLen(key)
}

func (e *MemoryEngine) HMSet(key string, fields map[string]string) {
	e.store.HMSet(key, fields)
}

func (e *MemoryEngine) HMGet(key string, fields ...string) []string {
	return e.store.HMGet(key, fields...)
}
