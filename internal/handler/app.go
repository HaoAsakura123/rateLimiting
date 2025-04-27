package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"ratelimiting/internal/storage"
)


func StatusBackendsHandle(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodGet{
		log.Println("INFO: uncorrect method")
	}

	url := r.URL.String()
	backendId := url[len("/status/"):]
	storage.BackendsLock.Lock()
	if val, ok := storage.Backends[backendId]; !ok{
		log.Printf("backend: %s was not found\n", backendId)
		http.Error(w, "Bad Request", http.StatusNotFound)
	} else{
		json.NewEncoder(w).Encode(val) 
	}
	storage.BackendsLock.Unlock()

}



// Обработчик добавления задачи
func AddTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Path[len("/addTask/"):]
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	newTask := storage.Task{
		TaskID:   taskID,
		TaskType: "from_http",
	}

	// Добавляем задачу в очередь
	storage.TasksLock.Lock()
	storage.Tasks = append(storage.Tasks, newTask)
	storage.TasksLock.Unlock()


	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "task accepted",
		"task_id": taskID,
	})
}


