package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

type activity struct {
	description string
	done        int
}

func main() {
	http.HandleFunc("/", handler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Fatal(http.ListenAndServe(":9999", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	fmt.Fprint(w, `<link rel="stylesheet" href="/static/style.css" type="text/css">`)

	fmt.Fprint(w, "<table>")
	activities := getActivities()
	for _, activity := range activities {
		fmt.Fprintf(w, `<tr><td>%s</td><td class="pct-done"><div class="meter"><span style="width: %d%%"></span></div></td></tr>`, activity.description, activity.done)
	}

	fmt.Fprint(w, "</table>")
}

func getActivities() []activity {
	var activities []activity

	db, err := sql.Open("postgres", "dbname=activities sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT description, ROUND(100.0 * points_done / points_total) "pct_done" FROM activities`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var description string
		var done int

		if err := rows.Scan(&description, &done); err != nil {
			log.Fatal(err)
		}
		activities = append(activities, activity{description: description, done: done})
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return activities
}
