package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	Backends     = make(map[string]Backend)
	BackendsLock = sync.RWMutex{}
	Tasks        = make([]Task, 0)
	TasksLock    = sync.RWMutex{}

	NumBack int
)

type Backend struct {
	ID           string    `json:"id"`
	CurrentTask  Task      `json:"task_id"`
	URL          string    `json:"url"`
	IsActive     bool      `json:"is_active"`
	Weight       int       `json:"weight"` // для улучшения алгоритма в последующем
	LastChecked  time.Time `json:"last_checked"`
	IsAvailiable bool      `json:"is_availible"` // работает или сломан
}

type Config struct {
	Backends []Backend `json:"backends"`
}

type Task struct {
	TaskID   string `json:"task_id"`
	TaskType string `json:"task_type"`
}
type DB struct {
	*sql.DB
}

type TaskRepository interface {
	SaveTask(task Task, backendID string) error
}

func (db *DB) SaveTask(task Task, backendID string) error {
	query := `INSERT INTO tasks (task_id, task_type, timestamp_at, backend_service) 
              VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(query, task.TaskID, task.TaskType, time.Now(), backendID)
	if err != nil {
		log.Printf("Failed to save task to DB: %v", err)
	}
	return err
}

// инициализация бд с последующими миграциями
func InitDB() (*DB, error) {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	user := os.Getenv("USER_DB")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable",
		host, port, user, password)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("error connecting to PostgreSQL: %v", err)
	}

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbname).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("error checking database existence: %v", err)
	}

	if !exists {

		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
		if err != nil {
			return nil, fmt.Errorf("error creating database: %v", err)
		}
		log.Printf("Database %s created", dbname)
	}

	db.Close()

	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging database: %v", err)
	}

	err = runMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("error running migrations: %v", err)
	}

	return &DB{db}, nil
}

func runMigrations(db *sql.DB) error {
	// Сначала проверяем/создаем таблицу для отслеживания миграций
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS migrations (
            name VARCHAR(255) PRIMARY KEY,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return fmt.Errorf("error creating migrations table: %v", err)
	}

	// Получаем список уже примененных миграций
	appliedMigrations := make(map[string]bool)
	rows, err := db.Query("SELECT name FROM migrations")
	if err != nil {
		return fmt.Errorf("error querying applied migrations: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("error scanning migration name: %v", err)
		}
		appliedMigrations[name] = true
	}

	// Определяем все миграции
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "enable_uuid_extension",
			sql:  `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		},
		{
			name: "create_tasks_table",
			sql: `
                CREATE TABLE IF NOT EXISTS tasks (
                    task_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                    task_type VARCHAR(100) NOT NULL,
                    timestamp_at TIMESTAMP WITH TIME ZONE NOT NULL,
                    backend_service VARCHAR(100) NOT NULL
                )
            `,
		},
	}

	// Применяем миграции
	for _, migration := range migrations {
		if !appliedMigrations[migration.name] {
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("error starting transaction for migration %s: %v", migration.name, err)
			}

			// Выполняем миграцию
			if _, err = tx.Exec(migration.sql); err != nil {
				tx.Rollback()
				return fmt.Errorf("error executing migration %s: %v", migration.name, err)
			}

			// Записываем факт применения миграции
			if _, err = tx.Exec("INSERT INTO migrations (name) VALUES ($1)", migration.name); err != nil {
				tx.Rollback()
				return fmt.Errorf("error recording migration %s: %v", migration.name, err)
			}

			// Коммитим транзакцию
			if err = tx.Commit(); err != nil {
				return fmt.Errorf("error committing migration %s: %v", migration.name, err)
			}

			log.Printf("Applied migration: %s", migration.name)
		}
	}

	return nil
}
