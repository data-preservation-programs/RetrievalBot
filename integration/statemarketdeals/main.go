package main

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/bcicen/jstream"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model/rpc"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klauspost/compress/zstd"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var logger = logging.Logger("state-market-deals")

func main() {
	ctx := context.Background()
	interval := env.GetDuration(env.StatemarketdealsInterval, 6*time.Hour)
	for {
		err := refresh(ctx)
		if err != nil {
			logger.Error(err)
		}

		logger.With("interval", interval).Info("sleeping")
		time.Sleep(interval)
	}
}

func refresh(ctx context.Context) error {
	batchSize := env.GetInt(env.StatemarketdealsBatchSize, 1000)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.StatemarketdealsMongoURI)))
	if err != nil {
		return errors.Wrap(err, "failed to connect to mongo")
	}

	//nolint:errcheck
	defer client.Disconnect(ctx)
	collection := client.Database(env.GetRequiredString(env.StatemarketdealsMongoDatabase)).
		Collection("state_market_deals")

	logger.Info("getting deal ids from mongo")
	dealIDCursor, err := collection.Find(ctx, bson.D{}, options.Find().SetProjection(bson.M{"deal_id": 1, "_id": 0}))
	if err != nil {
		return errors.Wrap(err, "failed to get deal ids")
	}

	defer dealIDCursor.Close(ctx)
	var dealIds []model.DealID
	err = dealIDCursor.All(ctx, &dealIds)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve all deal ids")
	}

	logger.Infof("retrieved %d deal ids", len(dealIds))
	dealIDSet := make(map[int32]struct{})
	for _, dealID := range dealIds {
		dealIDSet[dealID.DealID] = struct{}{}
	}

	logger.Info("getting deals from state market deals")
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		"https://marketdeals.s3.amazonaws.com/StateMarketDeals.json.zst",
		nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to make request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to get state market deals: %s", resp.Status)
	}

	decompressor, err := zstd.NewReader(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to create decompressor")
	}

	defer decompressor.Close()

	jsonDecoder := jstream.NewDecoder(decompressor, 1).EmitKV()
	count := 0
	dealBatch := make([]interface{}, 0, batchSize)
	for stream := range jsonDecoder.Stream() {
		keyValuePair, ok := stream.Value.(jstream.KV)

		if !ok {
			return errors.New("failed to get key value pair")
		}

		var deal rpc.Deal
		err = mapstructure.Decode(keyValuePair.Value, &deal)
		if err != nil {
			return errors.Wrap(err, "failed to decode deal")
		}

		// Skip the deal if the deal is not active yet
		if deal.State.SectorStartEpoch <= 0 {
			continue
		}

		// Skip the deal if the deal has already expired
		if model.EpochToTime(deal.Proposal.EndEpoch).Unix() <= time.Now().Unix() {
			continue
		}

		dealID, err := strconv.Atoi(keyValuePair.Key)
		if err != nil {
			return errors.Wrap(err, "failed to convert deal id to int")
		}

		// Insert into mongo if the deal is not in mongo
		//nolint:gosec
		if _, ok := dealIDSet[int32(dealID)]; !ok {
			dealState := model.DealState{
				//nolint:gosec
				DealID:     int32(dealID),
				PieceCID:   deal.Proposal.PieceCID.Root,
				Label:      deal.Proposal.Label,
				Verified:   deal.Proposal.VerifiedDeal,
				Client:     deal.Proposal.Client,
				Provider:   deal.Proposal.Provider,
				Expiration: model.EpochToTime(deal.Proposal.EndEpoch),
				PieceSize:  int64(deal.Proposal.PieceSize),
				Start:      model.EpochToTime(deal.State.SectorStartEpoch),
			}

			dealBatch = append(dealBatch, dealState)
			logger.With("deal_id", dealID).
				Debug("inserting deal state into mongo")

			if len(dealBatch) == batchSize {
				_, err := collection.InsertMany(ctx, dealBatch)
				if err != nil {
					return errors.Wrap(err, "failed to insert deal into mongo")
				}

				count += len(dealBatch)
				dealBatch = make([]interface{}, 0, batchSize)
			}
		}
	}

	if len(dealBatch) > 0 {
		_, err := collection.InsertMany(ctx, dealBatch)
		if err != nil {
			return errors.Wrap(err, "failed to insert deal into mongo")
		}

		count += len(dealBatch)
	}

	logger.With("count", count).Info("finished inserting deals into mongo")
	if jsonDecoder.Err() != nil {
		logger.With("position", jsonDecoder.Pos()).Warn("prematurely reached end of json stream")
		return errors.Wrap(jsonDecoder.Err(), "failed to decode json further")
	}

	// Finally, remove all expired deals from mongo
	deleteResult, err := collection.DeleteMany(ctx, bson.M{"expiration": bson.M{"$lt": time.Now()}})
	if err != nil {
		return errors.Wrap(err, "failed to delete expired deals")
	}

	logger.With("count", deleteResult.DeletedCount).Info("finished deleting expired deals from mongo")
	return nil
}
