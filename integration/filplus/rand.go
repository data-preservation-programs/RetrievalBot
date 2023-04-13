package main

import (
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"math"
	"math/rand"
	"time"
)

// RandomObjects Select l random objects from x with probability c^x[i].YearsTillExpiration().
func RandomObjects(x []model.DealState, l int, c float64) []model.DealState {
	// Initialize the random number generator.
	rand.Seed(time.Now().UnixNano())

	// Calculate the sum of C^age for all objects.
	var sum float64
	for _, obj := range x {
		sum += math.Pow(c, obj.YeasTillExpiration())
	}

	// Select Y random objects.
	selected := make(map[int32]bool)
	var results []model.DealState
	for i := 0; i < l; i++ {
		// Generate a random number between 0 and the sum.
		// nolint:gosec
		randNum := rand.Float64() * sum

		// Iterate over the objects and subtract C^age from randNum.
		for _, obj := range x {
			if selected[obj.DealID] {
				// Skip objects that have already been selected.
				continue
			}
			randNum -= math.Pow(c, obj.YeasTillExpiration())
			if randNum <= 0 {
				// Add the current object to the selected list.
				results = append(results, obj)
				selected[obj.DealID] = true
				break
			}
		}
	}

	return results
}
