package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const uuidForTests = "00000000-0000-0000-0000-000000000000"

func TestMain(m *testing.M) {
	var err error
	db, err = createDBConnection("habitcat_test")
	if err != nil {
		log.Fatalln("opening db connection failed:", err)
	}
	defer db.Close()

	os.Exit(m.Run())
}

func truncateDatabase() {
	db.Exec("TRUNCATE habit CASCADE")
	db.Exec("TRUNCATE goal CASCADE")
}

func createHabitProgress(id string, delta int, created *time.Time) {
	if created == nil {
		db.Exec("INSERT INTO habit_progress (habit_id, delta) VALUES ($1, $2)", id, delta)
	} else {
		db.Exec("INSERT INTO habit_progress (habit_id, delta, created) VALUES ($1, $2, $3)", id, delta, *created)
	}
}

func newHabit(description string, points int, period Period, start time.Time) *habit {
	return &habit{
		Description: description,
		Todo:        points,
		Period:      period,
		Start:       start,
	}
}

func TestGetHabitsNoProgress(t *testing.T) {
	createHabit(newHabit("Test", 1, PeriodWeek, time.Now()))
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
	id, _ := createHabit(newHabit("Test", 1, PeriodWeek, time.Now()))
	createHabitProgress(*id, 1, nil)
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
	id, _ := createHabit(newHabit("Test", 2, PeriodWeek, dt))
	createHabitProgress(*id, 1, &dt)
	createHabitProgress(*id, 1, &now)

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
	id, _ := createHabit(newHabit("Test", 1, PeriodWeek, time.Now()))
	defer truncateDatabase()

	habit, err := getHabit(*id)
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
	id, _ := createHabit(newHabit("Habit", 2, PeriodWeek, time.Now()))
	defer truncateDatabase()

	h, err := updateHabitProgress(*id)
	if err != nil {
		t.Errorf("Expected err to be nil, found %v", err)
	}
	if h.PctDone != 50 {
		t.Errorf("Expected PctDone to be 50, found %v", h.PctDone)
	}
	if h.Done != 1 {
		t.Errorf("Expected Done to be 1, found %v", h.Done)
	}
}

func TestUpdateHabitProgressNotFound(t *testing.T) {
	createHabit(newHabit("Habit", 2, PeriodWeek, time.Now()))
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
	id, _ := createHabit(newHabit("Habit", 2, PeriodWeek, time.Now()))
	defer truncateDatabase()

	url := "https://localhost/habits/" + *id
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitUpdateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %v", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected \"application/json\", got %v", w.Header().Get("Content-Type"))
	}
	var h habit
	err = json.Unmarshal(w.Body.Bytes(), &h)
	if err != nil {
		t.Error(err)
	}
	if h.PctDone != 50 {
		t.Errorf("Expected 50, got %v", h.PctDone)
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

func TestTotalPointsThisWeek(t *testing.T) {
	habits := []habit{
		habit{"id1", "description", 2, 1, 50, PeriodWeek, time.Now()},
	}
	done, todo := totalPointsThisWeek(habits)
	if done != 1 {
		t.Errorf("Expected 1, got %v", done)
	}
	if todo != 2 {
		t.Errorf("Expected 2, got %v", todo)
	}
}

func TestCurrentWeekNumber(t *testing.T) {
	dt := time.Date(2016, time.January, 11, 23, 0, 0, 0, time.UTC)
	week := currentWeekNumber(dt)
	if week != 2 {
		t.Errorf("Expected 2, got %v", week)
	}
}

func TestCreateHabitSuccessful(t *testing.T) {
	defer truncateDatabase()

	description := "description"
	todo := 33
	period := PeriodMonth
	start := time.Date(2016, time.January, 11, 0, 0, 0, 0, time.UTC)
	id, _ := createHabit(newHabit(description, todo, period, start))

	habit, err := getHabit(*id)
	if err != nil {
		t.Fatal(err)
	}
	if habit.Description != description {
		t.Errorf("Expected %v, got %v", description, habit.Description)
	}
	if habit.Todo != todo {
		t.Errorf("Expected %v, got %v", todo, habit.Todo)
	}
	if habit.Period != period {
		t.Errorf("Expected %v, got %v", period, habit.Period)
	}
	if habit.Start.Format("2006-01-11") != start.Format("2006-01-11") {
		t.Errorf("Expected %v, got %v", start.Format("2006-01-11"), habit.Start.Format("2006-01-11"))
	}
}

func TestCreateHabitFailure(t *testing.T) {
	defer truncateDatabase()

	description := "description"
	todo := 33
	period := Period("badperiod")
	start := time.Date(2016, time.January, 11, 0, 0, 0, 0, time.UTC)
	id, err := createHabit(newHabit(description, todo, period, start))
	if err == nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if id != nil {
		t.Errorf("Expected nil, got %v", id)
	}
}

func TestNewHabitHandlerSuccess(t *testing.T) {
	url := "https://localhost/habits/new"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitNewHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %v", w.Code)
	}
	expectedContentType := "text/html"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("Expected %v, got %v", expectedContentType, w.Header().Get("Content-Type"))
	}
}

func TestCreateHabitHandlerSuccess(t *testing.T) {
	url := "https://localhost/habits/create"
	description := "d"
	period := PeriodWeek
	todo := 33
	body := fmt.Sprintf("description=%s&period=%s&todo=%d", description, string(period), todo)
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	habitCreateHandler(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected %v, got %v", http.StatusFound, w.Code)
	}

	// make sure habit was created
	habits := getHabits()
	if len(habits) != 1 {
		t.Errorf("Expected 1 habit %d found", len(habits))
	}
	habit := habits[0]
	if habit.Description != description {
		t.Errorf("Expected %v, got %v", description, habit.Done)
	}
	if habit.Period != period {
		t.Errorf("Expected %v, got %v", period, habit.Period)
	}
	if habit.Todo != todo {
		t.Errorf("Expected %v, got %v", todo, habit.Todo)
	}
}

func TestCreateHabitHandlerWrongMethod(t *testing.T) {
	url := "https://localhost/habits/create"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	habitCreateHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected %v, got %v", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHabitHandlerSuccess(t *testing.T) {
	url := "https://localhost/habits"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	habitHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected %v, got %v", http.StatusOK, w.Code)
	}
	expectedContentType := "text/html"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("Expected %v, got %v", expectedContentType, w.Header().Get("Content-Type"))
	}
}
