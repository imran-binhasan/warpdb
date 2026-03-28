package store

// import necessery packages
import (
	"errors"
	"hash/fnv"
	"strconv"
	"sync"
)

const numShards = 256

type shard struct {
	data map[string]string
	mu sync.RWMutex
}

// store strcuture type
type Store struct {
	shards [numShards]shard
}

// create/set new key value
func (s *Store) Set(key, value string) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.data[key] = value
}

// for reusing of store without cloning in other modules
func NewStore() *Store {
	s := &Store{}
	for i := 0; i < numShards; i++ {
		s.shards[i].data = make(map[string]string)
	}
	return s
}

func (s * Store) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return &s.shards[h.Sum32()&(numShards-1)]
}

// exists method : checks if the value exists of a given key
func (s *Store) Exists(key string) bool {
    sh := s.getShard(key)
    sh.mu.RLock()
    defer sh.mu.RUnlock()
    _, exists := sh.data[key]
    return exists
}

// get method : retrieves the value for a given key
func (s *Store) Get(key string) (string, error) {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	value, exists := sh.data[key]
	if !exists {
		return "", errors.New("Key not found")
	}
	return value, nil
}

// del method : deletes the value for a given key
func (s *Store) Del(key string) error {
    sh := s.getShard(key)
    sh.mu.Lock()
    defer sh.mu.Unlock()
    _, exists := sh.data[key]
    if !exists {
        return errors.New("key not found")
    }
    delete(sh.data, key)
    return nil
}

// incr method : increment the value for a given key
func (s *Store) Incr(key string) (int, error) {
    sh := s.getShard(key)
    sh.mu.Lock()
    defer sh.mu.Unlock()
    value, exists := sh.data[key]
    if !exists {
        value = "0"
    }
    num, err := strconv.Atoi(value)
    if err != nil {
        return 0, errors.New("value is not an integer")
    }
    num++
    sh.data[key] = strconv.Itoa(num)
    return num, nil
}

// decr method : decriment the value for a given key
func (s *Store) Decr(key string) (int, error) {
    sh := s.getShard(key)
    sh.mu.Lock()
    defer sh.mu.Unlock()
    value, exists := sh.data[key]
    if !exists {
        value = "0"
    }
    num, err := strconv.Atoi(value)
    if err != nil {
        return 0, errors.New("value is not an integer")
    }
    num--
    sh.data[key] = strconv.Itoa(num)
    return num, nil
}
