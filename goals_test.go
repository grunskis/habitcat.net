package main

import "testing"

func TestGetGoalsSuccess(t *testing.T) {
	description := "doc"
	points := 15
	account, _ := CreateAccount("test@habitcat.net", "secret")
	createGoal(&goal{Description: description, PointsTotal: points}, account.Id)
	defer truncateDatabase()

	goals := getGoals(account.Id)
	if len(goals) != 1 {
		t.Errorf("Expected 1, got %v found", len(goals))
	}

	goal := goals[0]
	if goal.Description != description {
		t.Errorf("Expected %v, got %v", description, goal.Description)
	}
	if goal.PointsTotal != points {
		t.Errorf("Expected %v, got %v", points, goal.PointsDone)
	}
	if goal.PointsDone != 0 {
		t.Errorf("Expected 0, got %v", goal.PointsDone)
	}
	if goal.PctDone != 0 {
		t.Errorf("Expected 0%%, got %d%%", goal.PctDone)
	}
}
