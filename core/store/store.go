package store

import (
	"errors"
	"hash/fnv"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const numShards = 256

type shard struct {
	data    map[string]string
	expires map[string]time.Time
	lists   map[string][]string
	sets    map[string]map[string]struct{}
	hashes  map[string]map[string]string
	mu      sync.RWMutex
}

type Store struct {
	shards [numShards]shard
}

func (s *Store) Set(key, value string) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.data[key] = value
	delete(sh.expires, key)
}

func NewStore() *Store {
	s := &Store{}
	for i := 0; i < numShards; i++ {
		s.shards[i].data = make(map[string]string)
		s.shards[i].expires = make(map[string]time.Time)
		s.shards[i].lists = make(map[string][]string)
		s.shards[i].sets = make(map[string]map[string]struct{})
		s.shards[i].hashes = make(map[string]map[string]string)
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

func (s *Store) expireDataIfNeeded(sh *shard, key string) {
	if s.isExpired(sh, key) {
		delete(sh.data, key)
		delete(sh.expires, key)
		delete(sh.lists, key)
		delete(sh.sets, key)
		delete(sh.hashes, key)
	}
}

func (s *Store) Expire(key string, ttl time.Duration) error {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	_, hasStr := sh.data[key]
	_, hasList := sh.lists[key]
	_, hasSet := sh.sets[key]
	_, hasHash := sh.hashes[key]
	if !hasStr && !hasList && !hasSet && !hasHash {
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

func (s *Store) Exists(key string) bool {
	sh := s.getShard(key)

	sh.mu.RLock()
	_, exists := sh.data[key]
	_, hasList := sh.lists[key]
	_, hasSet := sh.sets[key]
	_, hasHash := sh.hashes[key]
	exp, hasExp := sh.expires[key]
	sh.mu.RUnlock()

	anyExists := exists || hasList || hasSet || hasHash

	if !anyExists {
		return false
	}

	if hasExp && time.Now().After(exp) {
		sh.mu.Lock()
		if exp2, still := sh.expires[key]; still && time.Now().After(exp2) {
			delete(sh.data, key)
			delete(sh.lists, key)
			delete(sh.sets, key)
			delete(sh.hashes, key)
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
	_, hasStr := sh.data[key]
	_, hasList := sh.lists[key]
	_, hasSet := sh.sets[key]
	_, hasHash := sh.hashes[key]
	if !hasStr && !hasList && !hasSet && !hasHash {
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
	_, hasStr := sh.data[key]
	_, hasList := sh.lists[key]
	_, hasSet := sh.sets[key]
	_, hasHash := sh.hashes[key]
	if !hasStr && !hasList && !hasSet && !hasHash {
		return errors.New("Key not found")
	}
	delete(sh.expires, key)
	return nil
}

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

func (s *Store) Del(key string) error {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	_, hasStr := sh.data[key]
	_, hasList := sh.lists[key]
	_, hasSet := sh.sets[key]
	_, hasHash := sh.hashes[key]
	if !hasStr && !hasList && !hasSet && !hasHash {
		return errors.New("key not found")
	}
	delete(sh.data, key)
	delete(sh.lists, key)
	delete(sh.sets, key)
	delete(sh.hashes, key)
	delete(sh.expires, key)
	return nil
}

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
					delete(sh.lists, key)
					delete(sh.sets, key)
					delete(sh.hashes, key)
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
		for key := range sh.data {
			exp, hasExp := sh.expires[key]
			if hasExp && now.After(exp) {
				continue
			}
			if matchPattern(pattern, key) {
				result = append(result, key)
			}
		}
		for key := range sh.lists {
			if _, skip := sh.data[key]; skip {
				continue
			}
			exp, hasExp := sh.expires[key]
			if hasExp && now.After(exp) {
				continue
			}
			if matchPattern(pattern, key) {
				result = append(result, key)
			}
		}
		for key := range sh.sets {
			if _, skip := sh.data[key]; skip {
				continue
			}
			exp, hasExp := sh.expires[key]
			if hasExp && now.After(exp) {
				continue
			}
			if matchPattern(pattern, key) {
				result = append(result, key)
			}
		}
		for key := range sh.hashes {
			if _, skip := sh.data[key]; skip {
				continue
			}
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

// --- List Operations ---

func (s *Store) LPush(key string, values ...string) int {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	list := sh.lists[key]
	n := make([]string, 0, len(values)+len(list))
	n = append(n, values...)
	n = append(n, list...)
	sh.lists[key] = n
	return len(n)
}

func (s *Store) RPush(key string, values ...string) int {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.lists[key] = append(sh.lists[key], values...)
	return len(sh.lists[key])
}

func (s *Store) LPop(key string) (string, error) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	list, ok := sh.lists[key]
	if !ok || len(list) == 0 {
		return "", errors.New("empty list")
	}
	val := list[0]
	sh.lists[key] = list[1:]
	if len(sh.lists[key]) == 0 {
		delete(sh.lists, key)
	}
	return val, nil
}

func (s *Store) RPop(key string) (string, error) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	list, ok := sh.lists[key]
	if !ok || len(list) == 0 {
		return "", errors.New("empty list")
	}
	last := len(list) - 1
	val := list[last]
	sh.lists[key] = list[:last]
	if len(sh.lists[key]) == 0 {
		delete(sh.lists, key)
	}
	return val, nil
}

func (s *Store) LLen(key string) int {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return len(sh.lists[key])
}

func (s *Store) LRange(key string, start, stop int) []string {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	list := sh.lists[key]
	if len(list) == 0 {
		return []string{}
	}
	llen := len(list)
	if start < 0 {
		start = llen + start
	}
	if stop < 0 {
		stop = llen + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= llen {
		stop = llen - 1
	}
	if start > stop {
		return []string{}
	}
	result := make([]string, stop-start+1)
	copy(result, list[start:stop+1])
	return result
}

func (s *Store) LIndex(key string, index int) (string, error) {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	list := sh.lists[key]
	if len(list) == 0 {
		return "", errors.New("empty list")
	}
	if index < 0 {
		index = len(list) + index
	}
	if index < 0 || index >= len(list) {
		return "", errors.New("index out of range")
	}
	return list[index], nil
}

// --- Set Operations ---

func (s *Store) SAdd(key string, members ...string) int {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	set, ok := sh.sets[key]
	if !ok {
		set = make(map[string]struct{})
		sh.sets[key] = set
	}
	added := 0
	for _, m := range members {
		if _, exists := set[m]; !exists {
			set[m] = struct{}{}
			added++
		}
	}
	return added
}

func (s *Store) SRem(key string, members ...string) int {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	set, ok := sh.sets[key]
	if !ok {
		return 0
	}
	removed := 0
	for _, m := range members {
		if _, exists := set[m]; exists {
			delete(set, m)
			removed++
		}
	}
	if len(set) == 0 {
		delete(sh.sets, key)
	}
	return removed
}

func (s *Store) SMembers(key string) []string {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	set := sh.sets[key]
	result := make([]string, 0, len(set))
	for m := range set {
		result = append(result, m)
	}
	return result
}

func (s *Store) SIsMember(key, member string) bool {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	_, ok := sh.sets[key][member]
	return ok
}

func (s *Store) SCard(key string) int {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return len(sh.sets[key])
}

func (s *Store) SPop(key string) (string, error) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	set := sh.sets[key]
	if len(set) == 0 {
		return "", errors.New("empty set")
	}
	for m := range set {
		delete(set, m)
		if len(set) == 0 {
			delete(sh.sets, key)
		}
		return m, nil
	}
	return "", errors.New("empty set")
}

func (s *Store) SRandMember(key string) (string, error) {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	set := sh.sets[key]
	if len(set) == 0 {
		return "", errors.New("empty set")
	}
	n := rand.Intn(len(set))
	i := 0
	for m := range set {
		if i == n {
			return m, nil
		}
		i++
	}
	return "", errors.New("empty set")
}

// --- Hash Operations ---

func (s *Store) HSet(key, field, value string) int {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	h, ok := sh.hashes[key]
	if !ok {
		h = make(map[string]string)
		sh.hashes[key] = h
	}
	_, existed := h[field]
	h[field] = value
	if existed {
		return 0
	}
	return 1
}

func (s *Store) HGet(key, field string) (string, error) {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	h, ok := sh.hashes[key]
	if !ok {
		return "", errors.New("field not found")
	}
	val, ok := h[field]
	if !ok {
		return "", errors.New("field not found")
	}
	return val, nil
}

func (s *Store) HDel(key string, fields ...string) int {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	h, ok := sh.hashes[key]
	if !ok {
		return 0
	}
	removed := 0
	for _, f := range fields {
		if _, exists := h[f]; exists {
			delete(h, f)
			removed++
		}
	}
	if len(h) == 0 {
		delete(sh.hashes, key)
	}
	return removed
}

func (s *Store) HGetAll(key string) []string {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	h := sh.hashes[key]
	result := make([]string, 0, len(h)*2)
	for k, v := range h {
		result = append(result, k, v)
	}
	return result
}

func (s *Store) HKeys(key string) []string {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	h := sh.hashes[key]
	result := make([]string, 0, len(h))
	for k := range h {
		result = append(result, k)
	}
	return result
}

func (s *Store) HVals(key string) []string {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	h := sh.hashes[key]
	result := make([]string, 0, len(h))
	for _, v := range h {
		result = append(result, v)
	}
	return result
}

func (s *Store) HExists(key, field string) bool {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	h, ok := sh.hashes[key]
	if !ok {
		return false
	}
	_, ok = h[field]
	return ok
}

func (s *Store) HLen(key string) int {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return len(sh.hashes[key])
}

func (s *Store) HMSet(key string, fields map[string]string) {
	sh := s.getShard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	h, ok := sh.hashes[key]
	if !ok {
		h = make(map[string]string)
		sh.hashes[key] = h
	}
	for k, v := range fields {
		h[k] = v
	}
}

func (s *Store) HMGet(key string, fields ...string) []string {
	sh := s.getShard(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	h := sh.hashes[key]
	result := make([]string, len(fields))
	for i, f := range fields {
		if v, ok := h[f]; ok {
			result[i] = v
		}
	}
	return result
}

func (s *Store) FlushAll() {
	for i := 0; i < numShards; i++ {
		sh := &s.shards[i]
		sh.mu.Lock()
		sh.data = make(map[string]string)
		sh.lists = make(map[string][]string)
		sh.sets = make(map[string]map[string]struct{})
		sh.hashes = make(map[string]map[string]string)
		sh.expires = make(map[string]time.Time)
		sh.mu.Unlock()
	}
}

func (s *Store) Size() int {
	total := 0
	for i := 0; i < numShards; i++ {
		sh := &s.shards[i]
		sh.mu.RLock()
		total += len(sh.data)
		total += len(sh.lists)
		total += len(sh.sets)
		total += len(sh.hashes)
		sh.mu.RUnlock()
	}
	return total
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
			end := -1
			for j := 1; j < len(pattern); j++ {
				if pattern[j] == ']' {
					end = j
					break
				}
			}
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
