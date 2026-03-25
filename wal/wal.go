package wal

// import (
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"sync"
// )

type Entry struct {
	Op string `json:"op"`
	Args []string `json:"args"`
}