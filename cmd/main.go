package main

import (
	"log"
	"net/http"
	"os"
	"ratelimiting/internal/handler"
	"ratelimiting/internal/logic"
	"ratelimiting/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	
	if err != nil {
	  log.Fatal("Error loading .env file")
	}

	db, err := storage.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	defer db.Close()

	PATH_TO_JSON := os.Getenv("PATH_TO_JSON")


	// инициализация из конфига бэкенд серверов
	servers, err := logic.LoadConfig(PATH_TO_JSON)
	if err != nil{
		log.Fatal(err)
	}
	for _, elem := range servers.Backends{
		storage.BackendsLock.Lock()
		storage.Backends[elem.ID] = storage.Backend{
			ID: elem.ID,
			URL: elem.URL,
			IsActive: elem.IsActive,
			Weight: elem.Weight,
		}	
		storage.BackendsLock.Unlock()
	}
	mux := http.NewServeMux()
	//Посмотреть информацию о конкретном backend сервере
	mux.HandleFunc("/status/", handler.StatusBackendsHandle)
	//Добавить новую задачу
	mux.HandleFunc("/newTask/", handler.AddTaskHandler)

	
	go logic.TaskGetter()// генерирует новые задичи раз в минуту
	go logic.ServeTask(db)// в горутине распределяет очередь задач по свободным бекэндам


	log.Println("INFO: server started at port :8080")
	http.ListenAndServe(":8080", mux)
}