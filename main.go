package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

type activity struct {
	Id          string
	Description string
	PctDone     int
}

var db *sql.DB

func main() {
	var err error
	db, err = createDBConnection()
	if err != nil {
		log.Fatalln("opening db connection failed:", err)
	}
	defer db.Close()

	http.HandleFunc("/", handler)
	http.HandleFunc("/update/", updateHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Fatal(http.ListenAndServe(":9999", nil))
}

func createDBConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", "dbname=activities user=postgres sslmode=disable")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	activities := getActivities()
	render(w, activities)
}

func render(w http.ResponseWriter, activities []activity) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(w, activities)
	if err != nil {
		log.Fatal(err)
	}
}

func getActivities() []activity {
	var activities []activity

	query := `SELECT id, description, ROUND(100.0 * points_done / points_total) FROM activities ORDER BY created`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id, description string
		var done int

		if err := rows.Scan(&id, &description, &done); err != nil {
			log.Fatal(err)
		}
		activities = append(activities, activity{Id: id, Description: description, PctDone: done})
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return activities
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", 405)
		return
	}
	activityUUID := r.URL.Path[len("/update/"):]
	if len(activityUUID) == 0 {
		http.Error(w, "", 400)
		return
	}

	newPct := updateActivityPoints(activityUUID)

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, fmt.Sprint(newPct))
}

func updateActivityPoints(activityUUID string) int {
	_, err := db.Exec("UPDATE activities SET points_done = points_done + 1 WHERE id = $1", activityUUID)
	if err != nil {
		log.Fatal(err)
	}

	var pctDone int
	row := db.QueryRow("SELECT ROUND(100.0 * points_done / points_total) FROM activities WHERE id = $1", activityUUID)
	if err := row.Scan(&pctDone); err != nil {
		log.Fatal(err)
	}
	log.Println("updated activity:", activityUUID, "new pct:", pctDone)
	return pctDone
}
