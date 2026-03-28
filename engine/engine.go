package engine

type StorageEngine interface {
	Get(key string) (string, error)
	Set(key, value string)  error
	Del(key string)  error
	Exists(key string) bool
	Incr(key string) (int, error)
	Decr(key string) (int, error)
}