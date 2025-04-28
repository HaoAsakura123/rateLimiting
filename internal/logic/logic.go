package logic

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"os"
	"ratelimiting/internal/storage"
	"time"

	"github.com/google/uuid"
)

func ServeTask(db *storage.DB){

	for{
		storage.TasksLock.Lock()
		length := len(storage.Tasks)
		storage.TasksLock.Unlock()
		if length > 0{
			storage.TasksLock.Lock()
			task := storage.Tasks[0]
			storage.Tasks = storage.Tasks[1:]
			storage.TasksLock.Unlock()
			for iter, elem := range storage.Backends{
				if !elem.IsActive {
					copIter := iter
					go Worker(db, copIter, task)
					break
				}
			}
		}
		// обрабатывает очередь запросов и запускает goroutine для свободных бэкендов
		time.Sleep(time.Second)
	}

}


func Worker(db *storage.DB, backID string, task storage.Task) {
    // Помечаем бэкенд как активный
    storage.BackendsLock.Lock()
    new := storage.Backends[backID]
    new.IsActive = true
    new.LastChecked = time.Now()
    storage.Backends[backID] = new
    storage.BackendsLock.Unlock()
    
    log.Printf("Task: %s started in backend server %s", task.TaskID, backID)
    
    // Имитация выполнения задачи
    time.Sleep(time.Duration(rand.IntN(50) + 1) * time.Second)
    
    // Помечаем бэкенд как неактивный
    storage.BackendsLock.Lock()
    new = storage.Backends[backID]
    new.IsActive = false
    new.LastChecked = time.Now()
    storage.Backends[backID] = new
    storage.BackendsLock.Unlock()
    
    log.Printf("Task: %s ended in backend server %s", task.TaskID, backID)
    
    // Сохраняем информацию о задаче в БД
	if err := db.SaveTask(task, backID); err != nil{
		log.Printf("Cannot save %s to db\n", task.TaskID)
	}
}


func TaskGetter(){
	for{
		time.Sleep(10 * time.Second)
		newTask := storage.Task{
			TaskID: TaskGenerator(),
			TaskType: "from_local",
		}
		storage.TasksLock.Lock()
		storage.Tasks = append(storage.Tasks, newTask)
		storage.TasksLock.Unlock()
	}
}

func TaskGenerator() string{
	return uuid.New().String()
}


func LoadConfig(path string) (*storage.Config, error) {
    file, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config storage.Config
    if err := json.Unmarshal(file, &config); err != nil {
        return nil, err
    }

    return &config, nil
}