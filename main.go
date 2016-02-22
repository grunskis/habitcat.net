package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	var err error
	db, err = createDBConnection("activities")
	if err != nil {
		log.Fatalln("opening db connection failed:", err)
	}
	defer db.Close()

	http.HandleFunc("/", goalHandler)
	http.HandleFunc("/update/", goalUpdateHandler)

	http.HandleFunc("/habits", habitHandler)
	http.HandleFunc("/habits/", habitUpdateHandler)

	staticFileServer := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	http.Handle("/static/", staticFileServer)
	http.Handle("/favicon.ico", staticFileServer)

	log.Println("Server listening on http://0.0.0.0:" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func createDBConnection(dbname string) (*sql.DB, error) {
	var dsn string
	_, found := os.LookupEnv("DATABASE_POSTGRESQL_USERNAME")
	if found {
		// for running test on semaphore ci
		dsn = fmt.Sprintf("dbname=%s user=runner password=semaphoredb sslmode=disable", dbname)
	} else {
		dsn, found = os.LookupEnv("DATABASE_URL")
		if !found {
			dsn = fmt.Sprintf("dbname=%s user=postgres sslmode=disable", dbname)
		}
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
