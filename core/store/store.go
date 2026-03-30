package store

// import necessery packages
import (
	"errors"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
	"time"
)

const numShards = 256

type shard struct {
	data    map[string]string
	expires map[string]time.Time
	mu      sync.RWMutex
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
	delete(sh.expires, key)
}

// for reusing of store without cloning in other modules
func NewStore() *Store {
	s := &Store{}
	for i := 0; i < numShards; i++ {
		s.shards[i].data = make(map[string]string)
		s.shards[i].expires = make(map[string]time.Time)
	}
	go s.activeExpiry()
	return s
}

func (s *Store) isExpired(sh *shard, key string) bool {
	exp, exists := sh.expires[key]
	if !exists {
		return false
	}
	return time.Now().After(exp)
}

func (s *Store) Expire(key string, ttl time.Duration) error {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if _, exists := sh.data[key]; !exists {
		return errors.New("Key not found")
	}
	sh.expires[key] = time.Now().Add(ttl)
	return nil
}

func (s *Store) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return &s.shards[h.Sum32()&(numShards-1)]
}

// exists method : checks if the value exists of a given key
func (s *Store) Exists(key string) bool {
	sh := s.getShard(key)

	sh.mu.RLock()
	_, exists := sh.data[key]
	exp, hasExp := sh.expires[key]
	sh.mu.RUnlock()

	if !exists {
		return false
	}

	if hasExp && time.Now().After(exp) {
		sh.mu.Lock()
		if exp2, still := sh.expires[key]; still && time.Now().After(exp2) {
			delete(sh.data, key)
			delete(sh.expires, key)
		}
		sh.mu.Unlock()
		return false
	}

	return true
}

func (s *Store) TTL(key string) (time.Duration, error) {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	if _, exists := sh.data[key]; !exists {
		return -2, nil
	}
	exp, hasExpiry := sh.expires[key]
	if !hasExpiry {
		return -1, nil
	}
	remaining := time.Until(exp)
	if remaining <= 0 {
		return -2, nil
	}
	return remaining, nil
}

func (s *Store) Persist(key string) error {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if _, exists := sh.data[key]; !exists {
		return errors.New("Key not found")
	}
	delete(sh.expires, key)
	return nil
}

// get method : retrieves the value for a given key
func (s *Store) Get(key string) (string, error) {
	sh := s.getShard(key)

	sh.mu.RLock()
	value, exists := sh.data[key]
	exp, hasExp := sh.expires[key]
	sh.mu.RUnlock()

	if !exists {
		return "", errors.New("key not found")
	}

	if hasExp && time.Now().After(exp) {
		sh.mu.Lock()
		delete(sh.data, key)
		delete(sh.expires, key)
		sh.mu.Unlock()
		return "", errors.New("key not found")
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
	delete(sh.expires, key)
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

func (s *Store) activeExpiry() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		for i := 0; i < numShards; i++ {
			sh := &s.shards[i]
			sh.mu.Lock()
			for key := range sh.expires {
				if time.Now().After(sh.expires[key]) {
					delete(sh.data, key)
					delete(sh.expires, key)
				}
			}
			sh.mu.Unlock()
		}
	}
}

func (s *Store) Keys(pattern string) []string {
	result := make([]string, 0, 64)
	now := time.Now()

	for i := 0; i < numShards; i++ {
		sh := &s.shards[i]
		sh.mu.RLock()
		for key, _ := range sh.data {
			exp, hasExp := sh.expires[key]
			if hasExp && now.After(exp) {
				continue
			}
			if matchPattern(pattern, key) {
				result = append(result, key)
			}
		}
		sh.mu.RUnlock()
	}

	return result
}

func matchPattern(pattern, key string) bool {
	if pattern == "*" {
		return true
	}
	return globMatch(pattern, key)
}

func globMatch(pattern, str string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			pattern = pattern[1:]
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(str); i++ {
				if globMatch(pattern, str[i:]) {
					return true
				}
			}
			return false

		case '?':
			if len(str) == 0 {
				return false
			}
			pattern = pattern[1:]
			str = str[1:]

		case '[':
			end := strings.Index(pattern, "]")
			if end < 0 {
				return pattern == str
			}
			if len(str) == 0 {
				return false
			}
			class := pattern[1:end]
			ch := str[0]
			matched := false
			for j := 0; j < len(class); j++ {
				if j+2 < len(class) && class[j+1] == '-' {
					if ch >= class[j] && ch <= class[j+2] {
						matched = true
					}
					j += 2
				} else if class[j] == ch {
					matched = true
				}
			}
			if !matched {
				return false
			}
			pattern = pattern[end+1:]
			str = str[1:]

		default:
			if len(str) == 0 || pattern[0] != str[0] {
				return false
			}
			pattern = pattern[1:]
			str = str[1:]
		}
	}
	return len(str) == 0
}

func (s *Store) FlushAll() {
	for i := 0; i < numShards; i++ {
		sh := &s.shards[i]
		sh.mu.Lock()
		sh.data = make(map[string]string)
		sh.expires = make(map[string]time.Time)
		sh.mu.Unlock()
	}
}
