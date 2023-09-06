package main

import (
	"testing"
	"time"

	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestWeight(t *testing.T) {
	now := model.TimeToEpoch(time.Now())
	objects := []model.DealState{
		{DealID: 1, SectorStart: now, PieceSize: 100, Client: "a"},
		{DealID: 2, SectorStart: now, PieceSize: 200, Client: "a"},
		{DealID: 3, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 100, Client: "a"},
		{DealID: 4, SectorStart: now, PieceSize: 100, Client: "b"},
		{DealID: 5, SectorStart: now, PieceSize: 100, Client: "c"},
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
		{DealID: 1, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 2, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 3, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 4, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 5, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 6, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 7, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 8, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 9, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 10, SectorStart: model.TimeToEpoch(time.Now()), PieceSize: 1, Client: "a"},
		{DealID: 11, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 12, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 13, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 14, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 15, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 16, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 17, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 18, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 19, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 20, SectorStart: model.TimeToEpoch(time.Now().Add(-24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 21, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 22, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 23, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 24, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 25, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 26, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 27, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 28, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 29, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
		{DealID: 30, SectorStart: model.TimeToEpoch(time.Now().Add(-2 * 24 * 365 * time.Hour)), PieceSize: 1, Client: "a"},
	}

	// Select 5 random objects with C = 2.
	selected := RandomObjects(objects, 15, 2.0, map[string]int64{"a": 1})

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
