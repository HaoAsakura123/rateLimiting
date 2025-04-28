package logic

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"os"
	"ratelimiting/internal/storage"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var lastIndex = 0

func ServeTask(db *storage.DB) {

	for {
		// я учитываю то что название бекэнд серверов формата backend{int}, int > 0

		storage.TasksLock.Lock()
		length := len(storage.Tasks)
		storage.TasksLock.Unlock()
		if length > 0 {
			storage.TasksLock.Lock()
			task := storage.Tasks[0]
			storage.Tasks = storage.Tasks[1:]
			storage.TasksLock.Unlock()
			for iter, elem := range storage.Backends {
				curInd, err := strconv.Atoi(iter[len("backend"):])
				if err != nil {
					continue
				}
				if !elem.IsAvailiable {
					if lastIndex == storage.NumBack {
						lastIndex = 0
					} else {
						lastIndex++
					}
				}
				if !elem.IsActive && curInd == lastIndex+1 {
					copIter := iter
					if lastIndex == storage.NumBack-1 {
						lastIndex = 0
					} else {
						lastIndex = curInd
					}
					if elem.IsAvailiable {
						go Worker(db, copIter, task)
						break
					} else {
						if lastIndex == storage.NumBack {
							lastIndex = 0
						} else {
							lastIndex++
						}
					}

				}
			}
			// for i:=1; i<=storage.NumBack; i++{
			// 	storage.BackendsLock.Lock()

			// 	curr := storage.Backends["backend"+strconv.Itoa(i)]

			// 	if !curr.IsActive && i == lastIndex + 1{
			// 		lastIndex = i
			// 	}

			// 	storage.BackendsLock.Unlock()
			// }
			//log.Printf("No availible backend service wait")
		}
		// обрабатывает очередь запросов и запускает goroutine для свободных бэкендов
		time.Sleep(time.Second)
	}

}

func Worker(db *storage.DB, backID string, task storage.Task) {
	// Помечаем бэкенд как активный
	storage.BackendsLock.Lock()
	new := storage.Backends[backID]
	if !new.IsAvailiable {
		return
	}
	new.IsActive = true
	new.LastChecked = time.Now()
	storage.Backends[backID] = new
	storage.BackendsLock.Unlock()

	log.Printf("Task: %s started in backend server %s", task.TaskID, backID)

	// Имитация выполнения задачи
	time.Sleep(time.Duration(rand.IntN(20)+1) * time.Second)

	storage.BackendsLock.Lock()
	check := storage.Backends[backID]
	storage.BackendsLock.Unlock()

	if !check.IsAvailiable {
		//тут я думал, куда же отправить задачу,на которой сломался бекэнд, и думаю ее самое место далеко сзади)
		storage.TasksLock.Lock()
		storage.Tasks = append(storage.Tasks, task)
		storage.TasksLock.Unlock()
		log.Printf("cannot end the task %s in %s\n", task.TaskID, backID)

	} else {

		// Помечаем бэкенд как неактивный
		storage.BackendsLock.Lock()
		new = storage.Backends[backID]
		new.IsActive = false
		new.LastChecked = time.Now()
		new.CurrentTask = task
		storage.Backends[backID] = new
		storage.BackendsLock.Unlock()
		log.Printf("Task: %s ended in backend server %s", task.TaskID, backID)

		// Сохраняем информацию о задаче в БД
		if err := db.SaveTask(task, backID); err != nil {
			log.Printf("Cannot save %s to db\n", task.TaskID)
		}
	}
}

func TaskGetter() {
	for {
		time.Sleep(5 * time.Second)
		newTask := storage.Task{
			TaskID:   TaskGenerator(),
			TaskType: "from_local",
		}
		storage.TasksLock.Lock()
		storage.Tasks = append(storage.Tasks, newTask)
		storage.TasksLock.Unlock()
	}
}

func TaskGenerator() string {
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

func BackendCrush() {
	for {
		time.Sleep(time.Duration(rand.IntN(10)+5) * time.Second)

		storage.BackendsLock.Lock()
		randBack := "backend" + strconv.Itoa((rand.IntN(storage.NumBack)))
		new := storage.Backends[randBack]
		storage.BackendsLock.Unlock()

		if !new.IsAvailiable {
			continue
		}

		new.IsAvailiable = false

		storage.BackendsLock.Lock()
		storage.Backends[randBack] = new
		storage.BackendsLock.Unlock()
		log.Printf("Backend service %s was broken\n", randBack)

	}
}

func BackendRecovery() {
	for {
		time.Sleep(3 * time.Second)

		for i := 1; i <= storage.NumBack; i++ {
			storage.BackendsLock.Lock()
			new := storage.Backends["backend"+strconv.Itoa(i)]
			storage.BackendsLock.Unlock()

			if !new.IsAvailiable {
				new.IsAvailiable = true
				log.Printf("Recovering backend service %s\n", new.ID)
				time.Sleep(5 * time.Second)

				storage.BackendsLock.Lock()
				storage.Backends["backend"+strconv.Itoa(i)] = new
				storage.BackendsLock.Unlock()
				log.Printf("backend service %s was recovered\n", new.ID)
			}

		}

	}
}
