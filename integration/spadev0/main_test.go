package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectCidsToTest(t *testing.T) {
	// Sample data for testing
	sampleData := map[int]ProviderReplicas{
		123: {size: 128, replicas: []Replica{
			{OptionalDagRoot: "root1", PieceCID: "cid1"},
		}},
		456: {size: 4096, replicas: []Replica{
			{OptionalDagRoot: "root3", PieceCID: "cid3"},
			{OptionalDagRoot: "root4", PieceCID: "cid4"},
			{OptionalDagRoot: "root5", PieceCID: "cid5"},
		}},
	}

	toTest := selectReplicasToTest(sampleData)

	// Ensure at least one replica is selected for each provider
	for providerID, replicas := range toTest {
		if len(replicas) == 0 {
			t.Errorf("No replicas selected for Provider %d", providerID)
		}
	}

	assert.Equal(t, 1, len(toTest[123]))
	assert.Equal(t, 2, len(toTest[456]))
}
