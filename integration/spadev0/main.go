package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

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
			&cli.StringFlag{
				Name:        "sources",
				DefaultText: "https://source-1/replicas.json.zst,https://source-2/replicas.json.zst",
				Usage:       "comma-separated list of sources to fetch replica list from",
				Required:    true,
			},
		},
		Action: func(cctx *cli.Context) error {
			// Pull CID list from Spade v0 endpoint, unmarshal

			// Select top level CIDs to begin sampling
			// * Balanced selection across SPs to ensure fair sampling

			// Retrieve top level CIDs to get 2nd layer

			// Select random CIDs from 2nd layer

			// Retrieve 2nd layer CIDs

			// Select random CIDs from 3rd layer
			// Add retrieval tasks to DB for a sampling of CIDs from the 3rd layer

			ctx := cctx.Context

			// Extract the sources from the flag
			sourcesStr := cctx.String("sources")
			sources := strings.Split(sourcesStr, ",")

			res, err := getCIDs(ctx, sources[0])

			if err != nil {
				return err
			}

			fmt.Println(res.StateEpoch)
			fmt.Println(res.ActiveReplicas[0].PieceCID)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func getCIDs(ctx context.Context, url string) (*ActiveReplicas, error) {
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

type ActiveReplicas struct {
	StateEpoch     uint            `json:"state_epoch"`
	ActiveReplicas []ActiveReplica `json:"active_replicas"`
}

type ActiveReplica struct {
	Contracts       []Contract `json:"contracts"`
	PieceCID        string     `json:"piece_cid"`
	PieceLog2Size   int        `json:"piece_log2_size"`
	OptionalDagRoot string     `json:"optional_dag_root"`
}

type Contract struct {
	ProviderID           int `json:"provider_id"`
	LegacyMarketID       int `json:"legacy_market_id"`
	LegacyMarketEndEpoch int `json:"legacy_market_end_epoch"`
}
