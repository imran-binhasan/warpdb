package store

// import necessery packages
import (
	"errors"
	"strconv"
	"sync"
)

// store strcuture type
type Store struct {
	data map[string]string
	mu sync.RWMutex
}

// create/set new key value
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// for reusing of store without cloning in other modules
func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

// exists method : checks if the value exists of a given key
func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[key]
	return exists
}

// get method : retrieves the value for a given key
func (s *Store) Get(key string) (string, error){
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	if !exists {
		return "", errors.New("key not found")
	}
	return  value, nil
}

// del method : deletes the value for a given key
func (s *Store) Del(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.data[key]
	if !exists {
		return errors.New("key not found")
	}
	delete(s.data, key)
	return nil
}

// incr method : increment the value for a given key
func (s *Store) Incr(key string) (int, error){
	s.mu.Lock()
	defer s.mu.Unlock()

	value,exists := s.data[key]
	if !exists {
		value = "0"
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("value is not an integer")
	}

	num++
	s.data[key] = strconv.Itoa(num)

	return num, nil
}

// decr method : decriment the value for a given key
func (s *Store) Decr(key string) (int, error){
	s.mu.Lock()
	defer s.mu.Unlock()

	value, exists := s.data[key]
	if !exists {
		value = "0"
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("value is not an integer")
	}

	num--
	s.data[key] = strconv.Itoa(num)
	return num, nil
}
