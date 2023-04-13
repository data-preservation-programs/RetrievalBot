package main

import (
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"testing"
	"time"
)

func TestRandomObjects(t *testing.T) {
	// Create a list of MyObject.
	objects := []model.DealState{
		{DealID: 1, Start: time.Now()},
		{DealID: 2, Start: time.Now()},
		{DealID: 3, Start: time.Now()},
		{DealID: 4, Start: time.Now()},
		{DealID: 5, Start: time.Now()},
		{DealID: 6, Start: time.Now()},
		{DealID: 7, Start: time.Now()},
		{DealID: 8, Start: time.Now()},
		{DealID: 9, Start: time.Now()},
		{DealID: 10, Start: time.Now()},
		{DealID: 11, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 12, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 13, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 14, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 15, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 16, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 17, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 18, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 19, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 20, Start: time.Now().Add(-24 * 365 * time.Hour)},
		{DealID: 21, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 22, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 23, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 24, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 25, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 26, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 27, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 28, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 29, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
		{DealID: 30, Start: time.Now().Add(-2 * 24 * 365 * time.Hour)},
	}

	// Select 5 random objects with C = 2.
	selected := RandomObjects(objects, 15, 2.0)

	// Check that the selected objects are distinct.
	selectedMap := make(map[int32]bool)
	for _, obj := range selected {
		if selectedMap[obj.DealID] {
			t.Errorf("Selected duplicate object with deal id %d", obj.DealID)
		}
		selectedMap[obj.DealID] = true
	}

	// Print the objects
	for _, obj := range selected {
		t.Logf("Selected object with deal id %d", obj.DealID)
	}
}
