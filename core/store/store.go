package store

import (
	"errors"
	"sync"
)

type Store struct {
	data map[string]string
	mu sync.RWMutex
}


func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *Store) Get(key string) (string, error){
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	if(!exists){
		return "", errors.New("Key not found")
	}
	return  value, nil
}

func (s *Store) Del(key string) ( error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.data[key]
	if(!exists){
		return errors.New("Key not found")
	}
	delete(s.data, key)
	return nil
}


func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}