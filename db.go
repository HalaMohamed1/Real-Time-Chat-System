package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() {
	var err error
	connStr := "host=localhost port=5432 user=postgres password=3242841 dbname=DB1 sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database connected")
}

func CreateUser(username, hashedPassword string) error {
	_, err := db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", username, hashedPassword)
	return err
}

func GetUserByUsername(username string) (*User, error) {
	row := db.QueryRow("SELECT id, username, password FROM users WHERE username = $1", username)

	user := &User{}
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, err
	}
	return user, nil
}
