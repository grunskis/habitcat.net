package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"
)

const (
	PeriodWeek  = "week"
	PeriodMonth = "month"
)

type habit struct {
	Id          string
	Description string
	Todo        int
	Done        int
	PctDone     int
	Modified    time.Time
}

func habitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/habits.html")
	if err != nil {
		log.Fatal(err)
	}

	habits := getHabits()
	err = t.Execute(w, habits)
	if err != nil {
		log.Fatal(err)
	}
}

func getHabits() []habit {
	query := `SELECT id,
                    description,
                    points,
                    (SELECT coalesce(sum(delta), 0)
                     FROM habit_progress p
                     WHERE h.id = p.habit_id
                       AND p.created >= date_trunc(h.period::text, now())
                       AND p.created < date_trunc(h.period::text, now()) + ('1 ' || h.period)::interval)
                   FROM habit h`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	var habits []habit
	var id, description string
	var done, todo int

	for rows.Next() {
		if err := rows.Scan(&id, &description, &todo, &done); err != nil {
			log.Fatal(err)
		}
		habits = append(habits, habit{
			Id:          id,
			Description: description,
			Todo:        todo,
			Done:        done,
			PctDone:     int(float64(done) / float64(todo) * 100),
		})
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return habits
}

func getHabit(uuid string) (*habit, error) {
	query := `SELECT id,
                    description,
                    points,
                    (SELECT coalesce(sum(delta), 0)
                     FROM habit_progress p
                     WHERE h.id = p.habit_id
                       AND p.created >= date_trunc(h.period::text, now())
                       AND p.created < date_trunc(h.period::text, now()) + ('1 ' || h.period)::interval)
                   FROM habit h
                   WHERE h.id = $1`

	var id, description string
	var done, todo int

	row := db.QueryRow(query, uuid)
	if err := row.Scan(&id, &description, &todo, &done); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("habit not found")
		} else {
			log.Fatal(err)
		}
	}

	return &habit{
		Id:          id,
		Description: description,
		Todo:        todo,
		Done:        done,
		PctDone:     int(float64(done) / float64(todo) * 100),
	}, nil
}

func habitUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	uuid := r.URL.Path[len("/habits/"):]
	if len(uuid) == 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	newPct, err := updateHabitProgress(uuid)
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	} else {
		fmt.Fprintf(w, fmt.Sprint(*newPct))
	}
}

func updateHabitProgress(uuid string) (*int, error) {
	habit, err := getHabit(uuid)
	if err != nil {
		return nil, errors.New("habit not found")
	}

	delta := 1
	_, err = db.Exec("INSERT INTO habit_progress (habit_id, delta) VALUES ($1, $2)", habit.Id, delta)
	if err != nil {
		log.Fatal(err)
	}

	newPctDone := int(float64(habit.Done+delta) / float64(habit.Todo) * 100)
	return &newPctDone, nil
}
