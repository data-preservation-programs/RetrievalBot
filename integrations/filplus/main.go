package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/common"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"strconv"
	"time"
)

func main() {
	logger := logging.Logger("filplus-integration")
	filplus := NewFilPlusIntegration()
	for {
		err := filplus.RunOnce(context.TODO())
		if err != nil {
			logger.Error(err)
		}

		time.Sleep(time.Minute)
	}
}

type FilPlusIntegration struct {
	taskCollection        *mongo.Collection
	marketDealsCollection *mongo.Collection
	batchSize             int
	requester             string
	locationResolver      common.LocationResolver
	providerResolver      common.ProviderResolver
	lotusAPI              api.FullNode
	lotusAPICloser        jsonrpc.ClientCloser
}

func NewFilPlusIntegration() *FilPlusIntegration {
	ctx := context.Background()
	taskClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("QUEUE_MONGO_URI")))
	if err != nil {
		panic(err)
	}
	taskCollection := taskClient.Database(os.Getenv("QUEUE_MONGO_DATABASE")).Collection("task_queue")

	stateMarketDealsClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("STATEMARKETDEALS_MONGO_URI")))
	if err != nil {
		panic(err)
	}
	marketDealsCollection := stateMarketDealsClient.Database(os.Getenv("STATEMARKETDEALS_MONGO_DATABASE")).
		Collection("state_market_deals")

	batchSizeStr := os.Getenv("FILPLUS_INTEGRATION_BATCH_SIZE")
	if batchSizeStr == "" {
		batchSizeStr = "100"
	}

	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		panic(err)
	}

	cacheTTLStr := os.Getenv("PROVIDER_CACHE_TTL_SECOND")
	if cacheTTLStr == "" {
		cacheTTLStr = "86400"
	}

	cacheTTL, err := strconv.Atoi(cacheTTLStr)
	if err != nil {
		panic(err)
	}

	locationResolver := common.NewLocationResolver(os.Getenv("IPINFO_TOKEN"), time.Duration(cacheTTL)*time.Second)
	providerResolver, err := common.NewProviderResolver(
		os.Getenv("LOTUS_API_URL"),
		os.Getenv("LOTUS_API_TOKEN"),
		time.Duration(cacheTTL)*time.Second)
	if err != nil {
		panic(err)
	}

	return &FilPlusIntegration{
		taskCollection:        taskCollection,
		marketDealsCollection: marketDealsCollection,
		batchSize:             batchSize,
		requester:             "filplus",
		locationResolver:      locationResolver,
		providerResolver:      *providerResolver,
	}
}

func (f *FilPlusIntegration) Close() {
	f.lotusAPICloser()
}

func (f *FilPlusIntegration) RunOnce(ctx context.Context) error {
	logger := logging.Logger("filplus-integration")

	// If the task queue already have batch size tasks, do nothing
	count, err := f.taskCollection.CountDocuments(ctx, bson.M{"requester": f.requester})
	if err != nil {
		return errors.Wrap(err, "failed to count tasks")
	}

	if count >= 3*int64(f.batchSize) {
		logger.Infof("task queue already have %d tasks, do nothing", f.batchSize)
		return nil
	}

	// Get random documents from state_market_deals that are still active and is verified
	aggregateResult, err := f.marketDealsCollection.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{
			"verified":   true,
			"Expiration": bson.M{"$gt": time.Now().Unix()},
		}},
		bson.M{"$sample": bson.M{"size": f.batchSize}},
	})

	if err != nil {
		return errors.Wrap(err, "failed to get sample documents")
	}

	var documents []common.DealState
	err = aggregateResult.All(ctx, &documents)
	if err != nil {
		return errors.Wrap(err, "failed to decode documents")
	}

	tasks := make([]interface{}, 0)
	// Insert the documents into task queue
	for _, document := range documents {
		// If the label is a correct CID, assume it is the payload CID and try GraphSync and Bitswap retrieval
		_, err := cid.Decode(document.Label)
		if err != nil {
			logger.With("label", document.Label, "deal_id", document.DealID).
				Debug("failed to decode label as CID")
			continue
		}

		providerInfo, err := f.providerResolver.ResolveProvider(ctx, document.Provider)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID).
				Error("failed to resolve provider")
			continue
		}

		location, err := f.locationResolver.ResolveMultiaddrsBytes(ctx, providerInfo.Multiaddrs)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID).
				Error("failed to resolve provider location")
			continue
		}

		for _, module := range []common.ModuleName{common.GraphSync, common.Bitswap} {
			tasks = append(tasks, common.Task{
				Requester: f.requester,
				Module:    module,
				Metadata: map[string]interface{}{
					"deal_id":       strconv.Itoa(int(document.DealID)),
					"client":        document.Client,
					"assume_label":  true,
					"retrieve_type": "root_block"},
				Provider: common.Provider{
					ID:        document.Provider,
					Country:   location.Country,
					Continent: location.Continent,
				},
				Content: common.Content{
					CID: document.Label,
				},
				CreatedAt: time.Now().UTC(),
			})
		}

		tasks = append(tasks, common.Task{
			Requester: f.requester,
			Module:    common.HTTP,
			Metadata: map[string]interface{}{
				"deal_id":       strconv.Itoa(int(document.DealID)),
				"client":        document.Client,
				"retrieve_type": "piece",
				"retrieve_size": 1 * 1024 * 1024},
			Provider: common.Provider{
				ID:        document.Provider,
				Country:   location.Country,
				Continent: location.Continent,
			},
			Content: common.Content{
				CID: document.PieceCID,
			},
			CreatedAt: time.Now().UTC(),
		})
	}

	insertResult, err := f.taskCollection.InsertMany(ctx, tasks)
	if err != nil {
		return errors.Wrap(err, "failed to insert tasks")
	}

	logger.Infof("inserted %d tasks", len(insertResult.InsertedIDs))
	return nil
}
