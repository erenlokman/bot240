package storage

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./crypto_news.db")
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}
	createTableSQL := `CREATE TABLE IF NOT EXISTS news (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        url TEXT NOT NULL,
        published_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`
	if _, err = db.Exec(createTableSQL); err != nil {
		log.Fatal("Error creating table: ", err)
	}
	return db
}
