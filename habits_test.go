package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

const uuidForTests = "00000000-0000-0000-0000-000000000000"

func TestMain(m *testing.M) {
	var err error
	db, err = createDBConnection("gandhi_test")
	if err != nil {
		log.Fatalln("opening db connection failed:", err)
	}
	defer db.Close()

	os.Exit(m.Run())
}

func truncateDatabase() {
	db.Exec("TRUNCATE habit CASCADE")
}

func createHabit(description string, points int, period string, start time.Time) string {
	var id string

	query := "INSERT INTO habit (description, points, period, start) VALUES ($1, $2, $3, $4) RETURNING id"
	err := db.QueryRow(query, description, points, period, start).Scan(&id)
	if err != nil {
		log.Fatal(err)
	}

	return id
}

func createHabitProgress(id string, delta int, created *time.Time) {
	if created == nil {
		db.Exec("INSERT INTO habit_progress (habit_id, delta) VALUES ($1, $2)", id, delta)
	} else {
		db.Exec("INSERT INTO habit_progress (habit_id, delta, created) VALUES ($1, $2, $3)", id, delta, *created)
	}
}

func TestGetHabitsNoProgress(t *testing.T) {
	createHabit("Test", 1, PeriodWeek, time.Now())
	defer truncateDatabase()

	habits := getHabits()
	if len(habits) != 1 {
		t.Errorf("Expected 1 habit %d found", len(habits))
	}

	habit := habits[0]
	if habit.Done != 0 {
		t.Errorf("Expected 0 points done %d found", habit.Done)
	}
	if habit.PctDone != 0 {
		t.Errorf("Expected 0%% done %d%% found", habit.PctDone)
	}
}

func TestGetHabitsWithProgress(t *testing.T) {
	id := createHabit("Test", 1, PeriodWeek, time.Now())
	createHabitProgress(id, 1, nil)
	defer truncateDatabase()

	habits := getHabits()
	if len(habits) != 1 {
		t.Errorf("Expected 1 habit %d found", len(habits))
	}

	habit := habits[0]
	if habit.Done != 1 {
		t.Errorf("Expected 1 point done %d found", habit.Done)
	}
	if habit.PctDone != 100 {
		t.Errorf("Expected 100%% done %d%% found", habit.PctDone)
	}
}

func TestGetHabitsWithProgressForCurrentPeriodOnly(t *testing.T) {
	now := time.Now()
	dt := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	id := createHabit("Test", 2, PeriodWeek, dt)
	createHabitProgress(id, 1, &dt)
	createHabitProgress(id, 1, &now)

	habits := getHabits()
	if len(habits) != 1 {
		t.Errorf("Expected 1 habit %d found", len(habits))
	}

	habit := habits[0]
	if habit.Done != 1 {
		t.Errorf("Expected 1 point done %d found", habit.Done)
	}
	if habit.PctDone != 50 {
		t.Errorf("Expected 50%% done %d%% found", habit.PctDone)
	}
}

func TestGetHabitExists(t *testing.T) {
	id := createHabit("Test", 1, PeriodWeek, time.Now())
	defer truncateDatabase()

	habit, err := getHabit(id)
	if err != nil {
		t.Errorf("Expecting err to be nil")
	}
	if habit.Description != "Test" {
		t.Errorf("Expected description to be Test but %s found", habit.Description)
	}
}

func TestGetHabitNotFound(t *testing.T) {
	habit, err := getHabit("00000000-0000-0000-0000-000000000000")
	if habit != nil {
		t.Errorf("Expected nil, found %v", habit)
	}
	if err.Error() != "habit not found" {
		t.Errorf("Expected \"habit not found\", found %v", err)
	}
}

func TestGetHabitError(t *testing.T) {
	// based on https://talks.golang.org/2014/testing.slide#23
	// since this is run in a separate process, lines touched by
	// this test will not appear in code coverage report

	if os.Getenv("BE_CRASHER") == "1" {
		getHabit("bad")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestGetHabitError")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestUpdateHabitProgressSuccess(t *testing.T) {
	id := createHabit("Habit", 2, PeriodWeek, time.Now())
	defer truncateDatabase()

	newPct, err := updateHabitProgress(id)
	if err != nil {
		t.Errorf("Expected err to be nil, found %v", err)
	}
	if *newPct != 50 {
		t.Errorf("Expected newPct to be 50, found %v", *newPct)
	}
}

func TestUpdateHabitProgressNotFound(t *testing.T) {
	createHabit("Habit", 2, PeriodWeek, time.Now())
	defer truncateDatabase()

	newPct, err := updateHabitProgress(uuidForTests)
	if newPct != nil {
		t.Errorf("Expected newPct to be nil, found %v", newPct)
	}
	if err.Error() != "habit not found" {
		t.Errorf("Expected \"habit not found\", found %v", err)
	}
}

func TestHabitUpdateHandlerSuccess(t *testing.T) {
	id := createHabit("Habit", 2, PeriodWeek, time.Now())
	defer truncateDatabase()

	url := "https://localhost/habits/" + id
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitUpdateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %v", w.Code)
	}
	if w.Body.String() != "50" {
		t.Errorf("Expected 50, got %v", w.Body.String())
	}
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected \"text/plain\", got %v", w.Header().Get("Content-Type"))
	}
}

func TestHabitUpdateHandlerNotFound(t *testing.T) {
	url := "https://localhost/habits/" + uuidForTests
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitUpdateHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %v", w.Code)
	}
}

func TestHabitUpdateHandlerWrongUUID(t *testing.T) {
	url := "https://localhost/habits/"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitUpdateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected %v, got %v", http.StatusNotFound, w.Code)
	}
}

func TestHabitUpdateHandlerWrongMethod(t *testing.T) {
	url := "https://localhost/habits/" + uuidForTests
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitUpdateHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected %v, got %v", http.StatusMethodNotAllowed, w.Code)
	}
}
