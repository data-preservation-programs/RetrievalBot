package main

import (
	"context"
	"strconv"
	"time"

	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/pkg/errors"
	"github.com/rjNemo/underscore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GroupID struct {
	Provider string `bson:"provider"`
	PieceCID string `bson:"piece_cid"`
}
type Row struct {
	ID       GroupID         `bson:"_id"`
	Document model.DealState `bson:"document"`
}

func AddSpadeTasks(ctx context.Context, requester string, replicasToTest map[int]ProviderReplicas) error {
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
	locationResolver := resolver.NewLocationResolver(env.GetRequiredString(env.IPInfoToken), locationCacheTTL)
	providerResolver, err := resolver.NewProviderResolver(
		env.GetString(env.LotusAPIUrl, "https://api.node.glif.io/rpc/v0"),
		env.GetString(env.LotusAPIToken, ""),
		providerCacheTTL)
	if err != nil {
		panic(err)
	}
	// Check public IP address
	ipInfo, err := resolver.GetPublicIPInfo(ctx, "", "")
	if err != nil {
		panic(err)
	}
	logger.With("ipinfo", ipInfo).Infof("Public IP info retrieved")

	// For each SPID, grab the market deals for its CIDs and then add tasks
	for spid, replica := range replicasToTest {
		// Get the relevant market deals for the given SP and replicas
		//nolint:govet
		pieceCids := underscore.Map(replica.replicas, func(r Replica) string {
			return r.PieceCID
		})

		result, err := marketDealsCollection.Aggregate(ctx, mongo.Pipeline{
			{{"$match", bson.D{
				{"provider", bson.D{{"$in", spid}}},
				{"piece_cid", bson.D{{"$in", pieceCids}}},
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

		prepareSpadeTasks(ctx, requester, ipInfo, documents, locationResolver, *providerResolver)
	}

	// TODO: Write the tasks out to database
	return nil
}

func prepareSpadeTasks(
	ctx context.Context,
	requester string,
	ipInfo resolver.IPInfo,
	documents []model.DealState,
	locationResolver resolver.LocationResolver,
	providerResolver resolver.ProviderResolver,
) (tasks []interface{}, results []interface{}) {
	for _, document := range documents {
		providerInfo, err := providerResolver.ResolveProvider(ctx, document.Provider)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID).
				Error("failed to resolve provider")
			continue
		}

		location, err := locationResolver.ResolveMultiaddrsBytes(ctx, providerInfo.Multiaddrs)
		if err != nil {
			if errors.As(err, &requesterror.BogonIPError{}) ||
				errors.As(err, &requesterror.InvalidIPError{}) ||
				errors.As(err, &requesterror.HostLookupError{}) ||
				errors.As(err, &requesterror.NoValidMultiAddrError{}) {

				// TODO: addErrorResults
				// results = addErrorResults(requester, ipInfo, results, document, providerInfo, location,
				// 	task.NoValidMultiAddrs, err.Error())
			} else {
				logger.With("provider", document.Provider, "deal_id", document.DealID, "err", err).
					Error("failed to resolve provider location")
			}
			continue
		}

		tasks = append(tasks, task.Task{
			Requester: requester,
			Module:    task.HTTP, // TODO: Bitswap
			Metadata: map[string]string{
				"deal_id":       strconv.Itoa(int(document.DealID)),
				"client":        document.Client,
				"retrieve_type": "spade",
				"retrieve_size": "1048576"},
			Provider: task.Provider{
				ID:         document.Provider,
				PeerID:     providerInfo.PeerId,
				Multiaddrs: convert.MultiaddrsBytesToStringArraySkippingError(providerInfo.Multiaddrs),
				City:       location.City,
				Region:     location.Region,
				Country:    location.Country,
				Continent:  location.Continent,
			},
			Content: task.Content{
				CID: document.PieceCID,
			},
			CreatedAt: time.Now().UTC(),
			Timeout:   env.GetDuration(env.FilplusIntegrationTaskTimeout, 15*time.Second),
		})
	}

	return tasks, results
}
