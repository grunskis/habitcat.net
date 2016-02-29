package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"
)

type Period string

const (
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

type habit struct {
	Id          string
	Description string
	Todo        int
	Done        int
	PctDone     int
	Period      Period
	Start       time.Time
}

func habitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/habits.html")
	if err != nil {
		log.Fatal(err)
	}

	habits := getHabits()
	done, todo := totalPointsThisWeek(habits)
	context := struct {
		Habits            []habit
		ThisWeekDone      int
		ThisWeekTodo      int
		CurrentWeekNumber int
		ThisWeekPctDone   int
	}{
		habits,
		done,
		todo,
		currentWeekNumber(time.Now()),
		int(float64(done) / float64(todo) * 100),
	}
	err = t.Execute(w, context)
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
                       AND p.created < date_trunc(h.period::text, now()) + ('1 ' || h.period)::interval),
                    period,
                    start
                  FROM habit h`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	var habits []habit
	var id, description, period string
	var done, todo int
	var start time.Time

	for rows.Next() {
		if err := rows.Scan(&id, &description, &todo, &done, &period, &start); err != nil {
			log.Fatal(err)
		}
		habits = append(habits, habit{
			Id:          id,
			Description: description,
			Todo:        todo,
			Done:        done,
			PctDone:     int(float64(done) / float64(todo) * 100),
			Period:      Period(period),
			Start:       start,
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
                       AND p.created < date_trunc(h.period::text, now()) + ('1 ' || h.period)::interval),
                    period,
                    start
                  FROM habit h
                  WHERE h.id = $1`

	var id, description, period string
	var done, todo int
	var start time.Time

	row := db.QueryRow(query, uuid)
	if err := row.Scan(&id, &description, &todo, &done, &period, &start); err != nil {
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
		Period:      Period(period),
		Start:       start,
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

	w.Header().Set("Content-Type", "application/json")

	h, err := updateHabitProgress(uuid)
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	} else {
		b, err := json.Marshal(h)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprint(w, string(b))
	}
}

func updateHabitProgress(uuid string) (*habit, error) {
	h, err := getHabit(uuid)
	if err != nil {
		return nil, errors.New("habit not found")
	}

	delta := 1
	_, err = db.Exec("INSERT INTO habit_progress (habit_id, delta) VALUES ($1, $2)", h.Id, delta)
	if err != nil {
		log.Fatal(err)
	}

	// TODO improve this
	h.Done = h.Done + delta
	h.PctDone = int(float64(h.Done) / float64(h.Todo) * 100)
	return h, nil
}

func totalPointsThisWeek(habits []habit) (int, int) {
	var todo, done int
	for _, h := range habits {
		done += h.Done
		todo += h.Todo
	}
	return done, todo
}

func currentWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func habitNewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("templates/habits_new.html")
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(w, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func habitCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	// TODO validation
	todo, err := strconv.Atoi(r.FormValue("todo"))
	if err != nil {
		log.Fatal(err)
	}
	period := Period(r.FormValue("period"))
	description := r.FormValue("description")

	newHabit := habit{
		Description: description,
		Period:      period,
		Todo:        todo,
	}
	createHabit(&newHabit)

	http.Redirect(w, r, "/habits", http.StatusFound)
}

func createHabit(h *habit) (*string, error) {
	var id string

	query := "INSERT INTO habit (description, points, period, start) VALUES ($1, $2, $3, $4) RETURNING id"
	err := db.QueryRow(query, h.Description, h.Todo, string(h.Period), h.Start).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
