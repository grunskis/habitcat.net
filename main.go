package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

type activity struct {
	Description string
	PctDone     int
}

func main() {
	http.HandleFunc("/", handler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Fatal(http.ListenAndServe(":9999", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	activities := getActivities()
	render(w, activities)
}

func render(w http.ResponseWriter, activities []activity) {
	const tpl = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <link rel="stylesheet" href="/static/style.css" type="text/css">
    <title>TODOs</title>
  </head>
  <body>
    <table>
      {{range .}}
      <tr>
        <td>{{.Description}}</td>
        <td class="pct-done">
          <div class="progress">
            <div style="width: {{.PctDone}}%"></div>
          </div>
        </td>
      </tr>
      {{end}}
    </table>
  </body>
</html>`

	w.Header().Set("Content-Type", "text/html")

	t, err := template.New("webpage").Parse(tpl)
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

	db, err := sql.Open("postgres", "dbname=activities user=postgres sslmode=disable")
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
		activities = append(activities, activity{Description: description, PctDone: done})
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return activities
}
