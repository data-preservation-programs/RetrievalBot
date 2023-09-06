package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/data-preservation-programs/RetrievalBot/integration/filplus/util"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model/rpc"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/data-preservation-programs/RetrievalBot/worker/bitswap"
	"github.com/data-preservation-programs/RetrievalBot/worker/graphsync"
	"github.com/data-preservation-programs/RetrievalBot/worker/http"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"github.com/ybbus/jsonrpc/v3"
)

//nolint:forbidigo,forcetypeassert,exhaustive
func main() {
	app := &cli.App{
		Name:      "oneoff",
		Usage:     "make a simple oneoff task that works with filplus tests",
		ArgsUsage: "providerID dealID",
		Action: func(cctx *cli.Context) error {
			ctx := cctx.Context
			providerID := cctx.Args().Get(0)
			dealIDStr := cctx.Args().Get(1)
			dealID, err := strconv.ParseUint(dealIDStr, 10, 32)
			if err != nil {
				return errors.Wrap(err, "failed to parse dealID")
			}
			providerResolver, err := resolver.NewProviderResolver(
				"https://api.node.glif.io/rpc/v0",
				"",
				time.Minute,
			)
			if err != nil {
				return errors.Wrap(err, "failed to create provider resolver")
			}

			providerInfo, err := providerResolver.ResolveProvider(ctx, providerID)
			if err != nil {
				return errors.Wrap(err, "failed to resolve provider")
			}

			locationResolver := resolver.NewLocationResolver("", time.Minute)
			_, err = locationResolver.ResolveMultiaddrsBytes(ctx, providerInfo.Multiaddrs)
			if err != nil {
				return errors.Wrap(err, "failed to resolve location")
			}

			ipInfo, err := resolver.GetPublicIPInfo(ctx, "", "")
			if err != nil {
				panic(err)
			}

			lotusClient := jsonrpc.NewClient("https://api.node.glif.io")
			var deal rpc.Deal
			err = lotusClient.CallFor(ctx, &deal, "Filecoin.StateMarketStorageDeal", dealID, nil)
			if err != nil {
				return errors.Wrap(err, "failed to get deal")
			}

			dealStates := []model.DealState{
				{
					DealID:      int32(dealID),
					PieceCID:    deal.Proposal.PieceCID.Root,
					PieceSize:   deal.Proposal.PieceSize,
					Label:       deal.Proposal.Label,
					Verified:    deal.Proposal.VerifiedDeal,
					Client:      deal.Proposal.Client,
					Provider:    deal.Proposal.Provider,
					Start:       deal.Proposal.StartEpoch,
					End:         deal.Proposal.EndEpoch,
					SectorStart: deal.State.SectorStartEpoch,
					Slashed:     deal.State.SlashEpoch,
					LastUpdated: deal.State.LastUpdatedEpoch,
				},
			}
			tasks, results := util.AddTasks(ctx, "oneoff", ipInfo, dealStates, locationResolver, *providerResolver)
			if len(results) > 0 {
				fmt.Println("Errors encountered when creating tasks:")
				for _, result := range results {
					r := result.(task.Result)
					fmt.Println(r)
				}
			}
			if len(tasks) > 0 {
				fmt.Println("Retrieval Test Results:")
				for _, tsk := range tasks {
					t := tsk.(task.Task)
					var result *task.RetrievalResult
					fmt.Printf(" -- Test %s --\n", t.Module)
					switch t.Module {
					case "graphsync":
						result, err = graphsync.Worker{}.DoWork(t)
					case "http":
						result, err = http.Worker{}.DoWork(t)
					case "bitswap":
						result, err = bitswap.Worker{}.DoWork(t)
					}
					if err != nil {
						fmt.Printf("Error: %s\n", err)
					} else {
						fmt.Printf("Success: %v\n", result)
					}
				}
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
