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

type goal struct {
	Id          string
	Description string
	PointsDone  int
	PointsTotal int
	PctDone     int
	Modified    time.Time
}

type byModified []goal

func (m byModified) Len() int           { return len(m) }
func (m byModified) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m byModified) Less(i, j int) bool { return (m[i].Modified).Before(m[j].Modified) }

func goalHandler(w http.ResponseWriter, r *http.Request) {
	var goalsInProgress, goalsDone []goal
	goals := getGoals()
	for _, g := range goals {
		if g.PctDone >= 100 {
			goalsDone = append(goalsDone, g)
		} else {
			goalsInProgress = append(goalsInProgress, g)
		}
	}
	context := struct {
		InProgress []goal
		Done       []goal
	}{
		goalsInProgress,
		goalsDone,
	}
	sort.Sort(sort.Reverse(byModified(context.InProgress)))
	sort.Sort(sort.Reverse(byModified(context.Done)))

	render(w, context)
}

func render(w http.ResponseWriter, goals interface{}) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/goals.html")
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(w, goals)
	if err != nil {
		log.Fatal(err)
	}
}

func getGoals() []goal {
	var goals []goal

	query := `SELECT id,
                         description,
                         ROUND(100.0 * points_done / points_total),
                         points_done,
                         points_total,
                         modified
                  FROM goal`

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
		goals = append(goals, goal{
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

	return goals
}

func goalUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	uuid := r.URL.Path[len("/goals/"):]
	if len(uuid) == 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	newPct := updateGoalPoints(uuid)

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, fmt.Sprint(newPct))
}

func updateGoalPoints(uuid string) int {
	_, err := db.Exec("UPDATE goal SET points_done = points_done + 1 WHERE id = $1", uuid)
	if err != nil {
		log.Fatal(err)
	}

	var pctDone int
	row := db.QueryRow("SELECT ROUND(100.0 * points_done / points_total) FROM goal WHERE id = $1", uuid)
	if err := row.Scan(&pctDone); err != nil {
		log.Fatal(err)
	}
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

	newGoal := goal{
		Description: description,
		PointsTotal: todo,
	}
	_, err = createGoal(&newGoal)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/goals", http.StatusFound)
}

func createGoal(g *goal) (*string, error) {
	var id string

	query := "INSERT INTO goal (description, points_total) VALUES ($1, $2) RETURNING id"
	err := db.QueryRow(query, g.Description, g.PointsTotal).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
