package main

import (
	"testing"
	"time"

	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestWeight(t *testing.T) {
	now := time.Now()
	objects := []model.DealState{
		{DealID: 1, Start: now, PieceSize: 100, Client: "a"},
		{DealID: 2, Start: now, PieceSize: 200, Client: "a"},
		{DealID: 3, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 100, Client: "a"},
		{DealID: 4, Start: now, PieceSize: 100, Client: "b"},
		{DealID: 5, Start: now, PieceSize: 100, Client: "c"},
	}
	clients := map[string]int64{
		"a": 16,
		"b": 1600,
		"c": 160000,
	}
	assert.InDelta(t, 25, weight(objects[0], 2, clients), 0.1)
	assert.InDelta(t, 50, weight(objects[1], 2, clients), 0.1)
	assert.InDelta(t, 12.5, weight(objects[2], 2, clients), 0.1)
	assert.InDelta(t, 2.5, weight(objects[3], 2, clients), 0.1)
	assert.InDelta(t, 0.25, weight(objects[4], 2, clients), 0.1)
}

func TestRandomObjects(t *testing.T) {
	// Create a list of MyObject.
	objects := []model.DealState{
		{DealID: 1, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 2, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 3, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 4, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 5, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 6, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 7, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 8, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 9, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 10, Start: time.Now(), PieceSize: 1, Client: "a"},
		{DealID: 11, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 12, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 13, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 14, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 15, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 16, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 17, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 18, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 19, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 20, Start: time.Now().Add(-24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 21, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 22, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 23, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 24, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 25, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 26, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 27, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 28, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 29, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
		{DealID: 30, Start: time.Now().Add(-2 * 24 * 365 * time.Hour), PieceSize: 1, Client: "a"},
	}

	// Select 5 random objects with C = 2.
	selected := RandomObjects(objects, 15, 2.0, map[string]int64{"a": 1})

	// Check that the selected objects are distinct.
	selectedMap := make(map[uint64]bool)
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
