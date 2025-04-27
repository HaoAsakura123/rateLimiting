package storage

import (
	"sync"
	"time"
)


var (
	Backends     = make(map[string] Backend)
	BackendsLock = sync.RWMutex{}
	Tasks = make([]Task, 0)
	TasksLock = sync.RWMutex{}
)



type Backend struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	IsActive     bool      `json:"is_active"`
	Weight       int       `json:"weight"` // для улучшения алгоритма в последующем
	LastChecked  time.Time `json:"last_checked"`
}

type Config struct {
    Backends []Backend `json:"backends"`
}

type Task struct {
	TaskID 	   string    `json:"task_id"`
	TaskType   string    `json:"task_type"`
}
