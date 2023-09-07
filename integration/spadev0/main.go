package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var logger = logging.Logger("spade-v0-tasks")

func main() {
	app := &cli.App{
		Name:  "spadev0",
		Usage: "run spade v0 CID sampling task",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "sources",
				DefaultText: "https://source-1/replicas.json.zst,https://source-2/replicas.json.zst",
				Usage:       "comma-separated list of sources to fetch replica list from",
				Required:    true,
			},
		},
		Action: func(cctx *cli.Context) error {
			ctx := cctx.Context

			// Extract the sources from the flag
			sources := cctx.StringSlice("sources")

			// TODO: support more than one - for now just takes the first one
			res, err := fetchActiveReplicas(ctx, sources[0])

			if err != nil {
				return err
			}

			var perProvider = make(map[int]ProviderReplicas)

			for _, replica := range res.ActiveReplicas {
				for _, contract := range replica.Contracts {
					providerID := contract.ProviderID
					size := 2 << replica.PieceLog2Size % 30 // Convert to GiB
					perProvider[providerID] = ProviderReplicas{
						size: perProvider[providerID].size + size,
						replicas: append(perProvider[providerID].replicas, Replica{
							PieceCID:        replica.PieceCID,
							PieceLog2Size:   replica.PieceLog2Size,
							OptionalDagRoot: replica.OptionalDagRoot,
						}),
					}
				}
			}

			replicasToTest := selectReplicasToTest(perProvider)

			for prov, rps := range replicasToTest {
				fmt.Printf("Provider %d will have %d tests\n", prov, len(rps))
			}

			AddSpadeTasks(ctx, "spadev0", replicasToTest)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func fetchActiveReplicas(ctx context.Context, url string) (*ActiveReplicas, error) {
	logger.Debug("fetching CIDs from %s", url)

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		url,
		nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to get spade CID list: %s", resp.Status)
	}

	decompressor, err := zstd.NewReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create decompressor")
	}

	defer decompressor.Close()

	data, err := ioutil.ReadAll(decompressor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read decompressed data")
	}

	// Unmarshal the JSON data into your struct
	var activeReplicas ActiveReplicas
	err = json.Unmarshal(data, &activeReplicas)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON data")
	}

	return &activeReplicas, nil
}

// Compute a number of CIDs to test, based on the total size of data (assuming in GiB)
// Minimum 1, then log2 of the size in TiB
// ex:
// < 4Tib = 1 cid
// 4 TiB - 16TiB = 2 cids
// 16 TiB - 256 TiB = 3 cids
func numCidsToTest(size int) int {
	return int(math.Max(math.Log2(float64(size/1024)), 1))
}

func selectReplicasToTest(perProvider map[int]ProviderReplicas) map[int][]Replica {
	var toTest = make(map[int][]Replica)

	for providerID, provider := range perProvider {
		toTest[providerID] = make([]Replica, 0)

		maxReplicas := len(provider.replicas)
		numCidsToTest := numCidsToTest(provider.size)

		// This condition should not happen, but just in case there's a situation
		// where a massive amount of bytes are being stored in relatively few CIDs
		if numCidsToTest > maxReplicas {
			numCidsToTest = maxReplicas
		}

		// Randomize which CIDs are selected
		rand.Seed(time.Now().UnixNano())
		indices := rand.Perm(maxReplicas)[:numCidsToTest]

		for _, index := range indices {
			toTest[providerID] = append(toTest[providerID], provider.replicas[index])
		}
	}

	return toTest
}

type ProviderReplicas struct {
	size     int
	replicas []Replica
}

type ActiveReplicas struct {
	StateEpoch     uint            `json:"state_epoch"`
	ActiveReplicas []ActiveReplica `json:"active_replicas"`
}

type ActiveReplica struct {
	Contracts []Contract `json:"contracts"`
	Replica
}

type Replica struct {
	PieceCID        string `json:"piece_cid"`
	PieceLog2Size   int    `json:"piece_log2_size"`
	OptionalDagRoot string `json:"optional_dag_root"`
}

type Contract struct {
	ProviderID           int `json:"provider_id"`
	LegacyMarketID       int `json:"legacy_market_id"`
	LegacyMarketEndEpoch int `json:"legacy_market_end_epoch"`
}
