package store

import (
	"testing"
	"strconv"
)


func TestSetGet(t *testing.T) {
	s := NewStore()
	s.Set("name", "Imran")
	val, err := s.Get("name")
	if err != nil {
		t.Fatal(err)
	}
	if val != "Imran" {
		t.Fatalf("expected Imran got %s", val)
	}
}

func TestGetMissing(t *testing.T) {
	s := NewStore()
	_, err := s.Get("ghost")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestIncr(t * testing.T) {
	s := NewStore()
	val, err := s.Incr("counter")
	if err != nil || val != 1 {
		t.Fatalf("expected 1 got %d err %v", val, err)
	}
	val, err = s.Incr("counter")
    if err != nil || val != 2 {
        t.Fatalf("expected 2 got %d err %v", val, err)
    }
}

func TestConcurrentIncr(t * testing.T) {
	s := NewStore()
	s.Set("counter", "0")
	done := make(chan bool)
	for i := 0; i < 1000; i++ {
		go func(){
			s.Incr("counter")
			done <- true
		}()
	}
	for i := 0; i < 1000; i++ {
		<- done
	}
	val, _ := s.Get("counter")
	if val != "1000" {
		t.Fatalf("expected value 1000 got %s", val)
	}
}

func BenchmarkSet(b *testing.B) {
    s := NewStore()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            s.Set(strconv.Itoa(i), "value")
            i++
        }
    })
}

func BenchmarkGet(b *testing.B) {
    s := NewStore()
    for i := 0; i < 1000; i++ {
        s.Set(strconv.Itoa(i), "value")
    }
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            s.Get(strconv.Itoa(i % 1000))
            i++
        }
    })
}

func BenchmarkIncr(b *testing.B) {
    s := NewStore()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            s.Incr("counter")
        }
    })
}