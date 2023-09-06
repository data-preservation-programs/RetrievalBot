package main

import (
	"math"
	"math/rand"

	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
)

func weight(obj model.DealState, c float64, totalPerClient map[string]int64) float64 {
	total, ok := totalPerClient[obj.Client]
	if !ok {
		return 0
	}
	return math.Pow(c, -obj.AgeInYears()) * float64(obj.PieceSize) / math.Sqrt(float64(total))
}

// RandomObjects Select l random objects from x with probability c^(-x[i].AgeInYears()).
func RandomObjects(x []model.DealState, l int, c float64, totalPerClient map[string]int64) []model.DealState {
	// Calculate the sum of C^age for all objects.
	var sum float64
	for _, obj := range x {
		sum += weight(obj, c, totalPerClient)
	}

	// Select Y random objects.
	selected := make(map[uint64]bool)
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
			randNum -= weight(obj, c, totalPerClient)
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
