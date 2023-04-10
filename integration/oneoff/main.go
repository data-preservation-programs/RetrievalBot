package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

var logger = logging.Logger("oneoff-integration")

func main() {
	app := &cli.App{
		Name:      "oneoff",
		Usage:     "make a simple oneoff task",
		ArgsUsage: "[module provider cid]",
		Action: func(cctx *cli.Context) error {
			ctx := context.Background()
			taskClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.QueueMongoURI)))
			if err != nil {
				return errors.Wrap(err, "Cannot connect to mongo")
			}
			taskCollection := taskClient.Database(env.GetRequiredString(env.QueueMongoDatabase)).Collection("task_queue")
			moduleName := cctx.Args().First()
			providerID := cctx.Args().Get(1)
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
			location, err := locationResolver.ResolveMultiaddrsBytes(ctx, providerInfo.Multiaddrs)
			if err != nil {
				return errors.Wrap(err, "failed to resolve location")
			}

			cidStr := cctx.Args().Get(2)
			tsk := task.Task{
				Requester: "oneoff",
				Module:    task.ModuleName(moduleName),
				Metadata:  nil,
				Provider: task.Provider{
					ID:         providerID,
					PeerID:     providerInfo.PeerId,
					Multiaddrs: convert.MultiaddrsBytesToStringArraySkippingError(providerInfo.Multiaddrs),
					City:       location.City,
					Region:     location.Region,
					Country:    location.Country,
					Continent:  location.Continent,
				},
				Content: task.Content{
					CID: cidStr,
				},
				CreatedAt: time.Now().UTC(),
				Timeout:   10 * time.Second,
			}
			insertResult, err := taskCollection.InsertOne(ctx, tsk)
			if err != nil {
				logger.Errorf("failed to insert task: %s", err)
			} else {
				logger.Infof("inserted task with id: %s", insertResult.InsertedID)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
