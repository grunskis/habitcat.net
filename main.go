package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

type activity struct {
	Id          string
	Description string
	PctDone     int
	Expires     *time.Time
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

	log.Println("Server listening on http://0.0.0.0:9999")
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
	now := time.Now()
	activities := getActivities()
	var inProgressActivities, doneOrExpiredActivities []activity
	for _, a := range activities {
		if a.PctDone >= 100 || (a.Expires != nil && now.After(*a.Expires)) {
			doneOrExpiredActivities = append(doneOrExpiredActivities, a)
		} else {
			inProgressActivities = append(inProgressActivities, a)
		}
	}
	context := struct {
		InProgress    []activity
		DoneOrExpired []activity
	}{
		inProgressActivities,
		doneOrExpiredActivities,
	}
	render(w, context)
}

func render(w http.ResponseWriter, activities interface{}) {
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

	query := `SELECT id, description, ROUND(100.0 * points_done / points_total), expires 
FROM activities 
ORDER BY created`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id, description string
		var done int
		var expires *time.Time

		if err := rows.Scan(&id, &description, &done, &expires); err != nil {
			log.Fatal(err)
		}
		activities = append(activities, activity{
			Id:          id,
			Description: description,
			PctDone:     done,
			Expires:     expires,
		})
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return activities
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	activityUUID := r.URL.Path[len("/update/"):]
	if len(activityUUID) == 0 {
		http.Error(w, "", http.StatusBadRequest)
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
