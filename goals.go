package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"text/template"
	"time"
)

type activity struct {
	Id          string
	Description string
	PointsDone  int
	PointsTotal int
	PctDone     int
	Modified    time.Time
}

type byModified []activity

func (m byModified) Len() int           { return len(m) }
func (m byModified) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m byModified) Less(i, j int) bool { return (m[i].Modified).Before(m[j].Modified) }

func goalHandler(w http.ResponseWriter, r *http.Request) {
	activities := getActivities()
	var inProgressActivities, doneActivities []activity
	for _, a := range activities {
		if a.PctDone >= 100 {
			doneActivities = append(doneActivities, a)
		} else {
			inProgressActivities = append(inProgressActivities, a)
		}
	}
	context := struct {
		InProgress []activity
		Done       []activity
	}{
		inProgressActivities,
		doneActivities,
	}
	sort.Sort(sort.Reverse(byModified(context.InProgress)))
	sort.Sort(sort.Reverse(byModified(context.Done)))
	render(w, context)
}

func render(w http.ResponseWriter, activities interface{}) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/goals.html")
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

	query := `SELECT id,
                         description,
                         ROUND(100.0 * points_done / points_total),
                         points_done,
                         points_total,
                         modified FROM activities`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id, description string
		var pctDone, pointsDone, pointsTotal int
		var modified time.Time

		if err := rows.Scan(&id, &description, &pctDone, &pointsDone, &pointsTotal, &modified); err != nil {
			log.Fatal(err)
		}
		activities = append(activities, activity{
			Id:          id,
			Description: description,
			PctDone:     pctDone,
			PointsDone:  pointsDone,
			PointsTotal: pointsTotal,
			Modified:    modified,
		})
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return activities
}

func goalUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	activityUUID := r.URL.Path[len("/goals/"):]
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

func goalNewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/goals_new.html")
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(w, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func goalCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	// TODO validation
	description := r.FormValue("description")
	todo, err := strconv.Atoi(r.FormValue("todo"))
	if err != nil {
		log.Fatal(err)
	}

	newActivity := activity{
		Description: description,
		PointsTotal: todo,
	}
	_, err = createGoal(&newActivity)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/goals", http.StatusFound)
}

func createGoal(a *activity) (*string, error) {
	var id string

	query := "INSERT INTO activities (description, points_total) VALUES ($1, $2) RETURNING id"
	err := db.QueryRow(query, a.Description, a.PointsTotal).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
