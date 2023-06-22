package main

import (
	"github.com/data-preservation-programs/RetrievalBot/integration/filplus/util"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"github.com/rjNemo/underscore"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

var logger = logging.Logger("spcoverage")

type GroupID struct {
	Provider string `bson:"provider"`
	PieceCID string `bson:"piece_cid"`
}
type Row struct {
	ID       GroupID         `bson:"_id"`
	Document model.DealState `bson:"document"`
}

func main() {
	app := &cli.App{
		Name:   "spcoverage",
		Usage:  "Send tasks to make sure all deals of a given SPs are covered",
		Action: run,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "sp",
				Usage:   "The SPs to be covered",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:     "requester",
				Usage:    "Name of the requester to tag the test result",
				Aliases:  []string{"r"},
				Required: true,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}

func run(c *cli.Context) error {
	ctx := c.Context
	sp := c.StringSlice("sp")
	requester := c.String("requester")
	if len(sp) == 0 {
		logger.Fatal("Please specify the SPs to be covered")
	}
	if requester == "" {
		logger.Fatal("Please specify the requester")
	}

	// Connect to the database
	stateMarketDealsClient, err := mongo.
		Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.StatemarketdealsMongoURI)))
	if err != nil {
		panic(err)
	}
	marketDealsCollection := stateMarketDealsClient.
		Database(env.GetRequiredString(env.StatemarketdealsMongoDatabase)).
		Collection("state_market_deals")

	providerCacheTTL := env.GetDuration(env.ProviderCacheTTL, 24*time.Hour)
	locationCacheTTL := env.GetDuration(env.LocationCacheTTL, 24*time.Hour)
	locationResolver := resolver.NewLocationResolver(env.GetRequiredString(env.IPInfoToken), providerCacheTTL)
	providerResolver, err := resolver.NewProviderResolver(
		env.GetString(env.LotusAPIUrl, "https://api.node.glif.io/rpc/v0"),
		env.GetString(env.LotusAPIToken, ""),
		locationCacheTTL)
	if err != nil {
		panic(err)
	}
	// Check public IP address
	ipInfo, err := resolver.GetPublicIPInfo(ctx, "", "")
	if err != nil {
		panic(err)
	}
	logger.With("ipinfo", ipInfo).Infof("Public IP info retrieved")

	// Get all CIDs for the given SPs
	//nolint:govet
	result, err := marketDealsCollection.Aggregate(ctx, mongo.Pipeline{
		{{"$match", bson.D{
			{"provider", bson.D{{"$in", sp}}},
			{"expiration", bson.D{{"$gt", time.Now()}}},
		}}},
		{{"$group", bson.D{
			{"_id", bson.D{{"provider", "$provider"}, {"piece_cid", "$piece_cid"}}},
			{"document", bson.D{{"$first", "$$ROOT"}}},
		}}},
	})
	if err != nil {
		return errors.Wrap(err, "failed to query market deals")
	}
	var rows []Row
	err = result.All(ctx, &rows)
	if err != nil {
		return errors.Wrap(err, "failed to decode market deals")
	}

	logger.Infow("Market deals retrieved", "count", len(rows))
	documents := underscore.Map(rows, func(row Row) model.DealState {
		return row.Document
	})
	tasks, results := util.AddTasks(ctx, requester, ipInfo, documents, locationResolver, *providerResolver)

	taskClient, err := mongo.
		Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.QueueMongoURI)))
	if err != nil {
		panic(err)
	}
	taskCollection := taskClient.
		Database(env.GetRequiredString(env.QueueMongoDatabase)).Collection("task_queue")

	if len(tasks) > 0 {
		_, err = taskCollection.InsertMany(ctx, tasks)
		if err != nil {
			return errors.Wrap(err, "failed to insert tasks")
		}
	}

	resultClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.ResultMongoURI)))
	if err != nil {
		panic(err)
	}
	resultCollection := resultClient.
		Database(env.GetRequiredString(env.ResultMongoDatabase)).
		Collection("task_result")

	if len(results) > 0 {
		_, err = resultCollection.InsertMany(ctx, results)
		if err != nil {
			return errors.Wrap(err, "failed to insert results")
		}
	}

	return nil
}
