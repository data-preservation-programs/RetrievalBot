package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
	"strconv"
	"time"
)

var logger = logging.Logger("filplus-integration")

func main() {
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
	resultCollection      *mongo.Collection
	batchSize             int
	requester             string
	locationResolver      resolver.LocationResolver
	providerResolver      resolver.ProviderResolver
	ipInfo                resolver.IPInfo
}

func NewFilPlusIntegration() *FilPlusIntegration {
	ctx := context.Background()
	taskClient, err := mongo.
		Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.QueueMongoURI)))
	if err != nil {
		panic(err)
	}
	taskCollection := taskClient.
		Database(env.GetRequiredString(env.QueueMongoDatabase)).Collection("task_queue")

	stateMarketDealsClient, err := mongo.
		Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.StatemarketdealsMongoURI)))
	if err != nil {
		panic(err)
	}
	marketDealsCollection := stateMarketDealsClient.
		Database(env.GetRequiredString(env.StatemarketdealsMongoDatabase)).
		Collection("state_market_deals")

	resultClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.ResultMongoURI)))
	if err != nil {
		panic(err)
	}
	resultCollection := resultClient.
		Database(env.GetRequiredString(env.ResultMongoDatabase)).
		Collection("task_result")

	batchSize := env.GetInt(env.FilplusIntegrationBatchSize, 100)
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
	ipInfo, err := resolver.GetPublicIPInfo(context.TODO(), "", "")
	if err != nil {
		panic(err)
	}

	logger.With("ipinfo", ipInfo).Infof("Public IP info retrieved")

	return &FilPlusIntegration{
		taskCollection:        taskCollection,
		marketDealsCollection: marketDealsCollection,
		batchSize:             batchSize,
		requester:             "filplus",
		locationResolver:      locationResolver,
		providerResolver:      *providerResolver,
		resultCollection:      resultCollection,
		ipInfo:                ipInfo,
	}
}

var moduleMetadataMap = map[task.ModuleName]map[string]string{
	task.GraphSync: {
		"assume_label":  "true",
		"retrieve_type": "root_block",
	},
	task.Bitswap: {
		"assume_label":  "true",
		"retrieve_type": "root_block",
	},
	task.HTTP: {
		"retrieve_type": "piece",
		"retrieve_size": "1048576",
	},
}

func (f *FilPlusIntegration) addErrorResults(results []interface{}, document model.DealState,
	providerInfo resolver.MinerInfo, location resolver.IPInfo,
	errorCode task.ErrorCode, errorMessage string) []interface{} {
	for module, metadata := range moduleMetadataMap {
		newMetadata := make(map[string]string)
		for k, v := range metadata {
			newMetadata[k] = v
		}
		newMetadata["deal_id"] = strconv.Itoa(int(document.DealID))
		newMetadata["client"] = document.Client
		results = append(results, task.Result{
			Task: task.Task{
				Requester: f.requester,
				Module:    module,
				Metadata:  newMetadata,
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
					CID: document.Label,
				},
				CreatedAt: time.Now().UTC(),
				Timeout:   env.GetDuration(env.FilplusIntegrationTaskTimeout, 15*time.Second)},
			Retriever: task.Retriever{
				PublicIP:  f.ipInfo.IP,
				City:      f.ipInfo.City,
				Region:    f.ipInfo.Region,
				Country:   f.ipInfo.Country,
				Continent: f.ipInfo.Continent,
				ASN:       f.ipInfo.ASN,
				ISP:       f.ipInfo.ISP,
				Latitude:  f.ipInfo.Latitude,
				Longitude: f.ipInfo.Longitude,
			},
			Result: task.RetrievalResult{
				Success:      false,
				ErrorCode:    errorCode,
				ErrorMessage: errorMessage,
				TTFB:         0,
				Speed:        0,
				Duration:     0,
				Downloaded:   0,
			},
			CreatedAt: time.Now().UTC(),
		})
	}
	return results
}

func (f *FilPlusIntegration) RunOnce(ctx context.Context) error {
	logger.Info("start running filplus integration")

	// If the task queue already have batch size tasks, do nothing
	count, err := f.taskCollection.CountDocuments(ctx, bson.M{"requester": f.requester})
	if err != nil {
		return errors.Wrap(err, "failed to count tasks")
	}

	if count >= 3*int64(f.batchSize) {
		logger.Infof("task queue already have %d tasks, do nothing", f.batchSize*3)
		return nil
	}

	// Get random documents from state_market_deals that are still active and is verified
	aggregateResult, err := f.marketDealsCollection.Aggregate(ctx, bson.A{
		bson.M{"$sample": bson.M{"size": f.batchSize}},
		bson.M{"$match": bson.M{
			"verified":   true,
			"expiration": bson.M{"$gt": time.Now().UTC()},
		}},
	})

	if err != nil {
		return errors.Wrap(err, "failed to get sample documents")
	}

	var documents []model.DealState
	err = aggregateResult.All(ctx, &documents)
	if err != nil {
		return errors.Wrap(err, "failed to decode documents")
	}

	tasks := make([]interface{}, 0)
	results := make([]interface{}, 0)
	// Insert the documents into task queue
	for _, document := range documents {
		// If the label is a correct CID, assume it is the payload CID and try GraphSync and Bitswap retrieval
		labelCID, err := cid.Decode(document.Label)
		if err != nil {
			logger.With("label", document.Label, "deal_id", document.DealID).
				Debug("failed to decode label as CID")
			continue
		}

		isPayloadCID := true
		// Skip graphsync and bitswap if the cid is not decodable, i.e. it is a pieceCID
		if !slices.Contains([]uint64{cid.Raw, cid.DagCBOR, cid.DagProtobuf, cid.DagJSON, cid.DagJOSE},
			labelCID.Prefix().Codec) {
			logger.With("provider", document.Provider, "deal_id", document.DealID,
				"label", document.Label, "codec", labelCID.Prefix().Codec).
				Info("Skip Bitswap and Graphsync because the Label is likely not a payload CID")
			isPayloadCID = false
		}

		providerInfo, err := f.providerResolver.ResolveProvider(ctx, document.Provider)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID).
				Error("failed to resolve provider")
			continue
		}

		location, err := f.locationResolver.ResolveMultiaddrsBytes(ctx, providerInfo.Multiaddrs)
		if err != nil {
			if errors.As(err, &requesterror.BogonIPError{}) ||
				errors.As(err, &requesterror.InvalidIPError{}) ||
				errors.As(err, &requesterror.HostLookupError{}) ||
				errors.As(err, &requesterror.NoValidMultiAddrError{}) {
				results = f.addErrorResults(results, document, providerInfo, location,
					task.NoValidMultiAddrs, err.Error())
			} else {
				logger.With("provider", document.Provider, "deal_id", document.DealID).
					Error("failed to resolve provider location")
			}
			continue
		}

		_, err = peer.Decode(providerInfo.PeerId)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID, "peerID", providerInfo.PeerId,
				"err", err).
				Info("failed to decode peerID")
			results = f.addErrorResults(results, document, providerInfo, location,
				task.InvalidPeerID, err.Error())
		}

		if isPayloadCID {
			for _, module := range []task.ModuleName{task.GraphSync, task.Bitswap} {
				tasks = append(tasks, task.Task{
					Requester: f.requester,
					Module:    module,
					Metadata: map[string]string{
						"deal_id":       strconv.Itoa(int(document.DealID)),
						"client":        document.Client,
						"assume_label":  "true",
						"retrieve_type": "root_block"},
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
						CID: document.Label,
					},
					CreatedAt: time.Now().UTC(),
					Timeout:   env.GetDuration(env.FilplusIntegrationTaskTimeout, 15*time.Second),
				})
			}
		}

		tasks = append(tasks, task.Task{
			Requester: f.requester,
			Module:    task.HTTP,
			Metadata: map[string]string{
				"deal_id":       strconv.Itoa(int(document.DealID)),
				"client":        document.Client,
				"retrieve_type": "piece",
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

	if len(tasks) > 0 {
		_, err = f.taskCollection.InsertMany(ctx, tasks)
		if err != nil {
			return errors.Wrap(err, "failed to insert tasks")
		}
	}

	logger.With("count", len(tasks)).Info("inserted tasks")

	if len(results) > 0 {
		_, err = f.resultCollection.InsertMany(ctx, results)
		if err != nil {
			return errors.Wrap(err, "failed to insert results")
		}
	}

	logger.With("count", len(results)).Info("inserted results")

	return nil
}
