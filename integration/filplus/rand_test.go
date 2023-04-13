package main

import (
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"testing"
	"time"
)

func TestRandomObjects(t *testing.T) {
	// Create a list of MyObject.
	objects := []model.DealState{
		{DealID: 1, Expiration: time.Now()},
		{DealID: 2, Expiration: time.Now()},
		{DealID: 3, Expiration: time.Now()},
		{DealID: 4, Expiration: time.Now()},
		{DealID: 5, Expiration: time.Now()},
		{DealID: 6, Expiration: time.Now()},
		{DealID: 7, Expiration: time.Now()},
		{DealID: 8, Expiration: time.Now()},
		{DealID: 9, Expiration: time.Now()},
		{DealID: 10, Expiration: time.Now()},
		{DealID: 11, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 12, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 13, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 14, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 15, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 16, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 17, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 18, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 19, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 20, Expiration: time.Now().Add(24 * 365 * time.Hour)},
		{DealID: 21, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 22, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 23, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 24, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 25, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 26, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 27, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 28, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 29, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
		{DealID: 30, Expiration: time.Now().Add(2 * 24 * 365 * time.Hour)},
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
