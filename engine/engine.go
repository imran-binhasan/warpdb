package engine

type StorageEngine interface {
	Get(key string) (string, error)
	Set(key, value string)  error
	Del(key string)  error
	Exists(key string) bool
}