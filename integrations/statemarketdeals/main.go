package main

import (
	"context"
	"github.com/bcicen/jstream"
	"github.com/data-preservation-programs/RetrievalBot/common"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klauspost/compress/zstd"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Deal struct {
	Proposal DealProposal
	State    DealState
}

type Cid struct {
	Root string `json:"/" mapstructure:"/"`
}

type DealProposal struct {
	PieceCID     Cid
	PieceSize    uint64
	VerifiedDeal bool
	Client       string
	Provider     string
	Label        string
	StartEpoch   int32
	EndEpoch     int32
}

type DealState struct {
	SectorStartEpoch int32
	LastUpdatedEpoch int32
	SlashEpoch       int32
}

func epochToTime(epoch int32) time.Time {
	return time.Unix(int64(epoch*30+1598306400), 0)
}

func main() {
	ctx := context.Background()
	for {
		err := refresh(ctx)
		if err != nil {
			logging.Logger("state-market-deals").Error(err)
		}

		logging.Logger("state-market-deals").Info("sleeping for 6 hours")
		time.Sleep(6 * time.Hour)
	}
}

func refresh(ctx context.Context) error {
	logger := logging.Logger("state-market-deals")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("STATEMARKETDEALS_MONGO_URI")))
	if err != nil {
		return errors.Wrap(err, "failed to connect to mongo")
	}

	defer client.Disconnect(ctx)
	collection := client.Database(os.Getenv("STATEMARKETDEALS_MONGO_DATABASE")).
		Collection("state_market_deals")

	logger.Info("getting deal ids from mongo")
	dealIdCursor, err := collection.Find(ctx, bson.D{}, options.Find().SetProjection(bson.M{"deal_id": 1, "_id": 0}))
	if err != nil {
		return errors.Wrap(err, "failed to get deal ids")
	}

	defer dealIdCursor.Close(ctx)
	var dealIds []common.DealID
	err = dealIdCursor.All(ctx, &dealIds)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve all deal ids")
	}

	logger.Infof("retrieved %d deal ids", len(dealIds))
	dealIdSet := make(map[int32]struct{})
	for _, dealId := range dealIds {
		dealIdSet[dealId.DealID] = struct{}{}
	}

	logger.Info("getting deals from state market deals")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://marketdeals.s3.amazonaws.com/StateMarketDeals.json.zst", nil)
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
	for stream := range jsonDecoder.Stream() {
		keyValuePair, ok := stream.Value.(jstream.KV)
		if !ok {
			return errors.New("failed to get key value pair")
		}

		var deal Deal
		err = mapstructure.Decode(keyValuePair.Value, &deal)
		if err != nil {
			return errors.Wrap(err, "failed to decode deal")
		}

		// Skip the deal if the deal is not active yet
		if deal.State.SectorStartEpoch <= 0 {
			continue
		}

		dealId, err := strconv.Atoi(keyValuePair.Key)
		if err != nil {
			return errors.Wrap(err, "failed to convert deal id to int")
		}

		// Insert into mongo if the deal is not in mongo
		if _, ok := dealIdSet[int32(dealId)]; !ok {
			dealState := common.DealState{
				DealID:     int32(dealId),
				PieceCID:   deal.Proposal.PieceCID.Root,
				Label:      deal.Proposal.Label,
				Verified:   deal.Proposal.VerifiedDeal,
				Client:     deal.Proposal.Client,
				Provider:   deal.Proposal.Provider,
				Expiration: epochToTime(deal.Proposal.EndEpoch),
			}

			inserted, err := collection.InsertOne(ctx, dealState)
			if err != nil {
				return errors.Wrap(err, "failed to insert deal into mongo")
			}

			logger.With("deal_id", dealId).
				With("inserted_id", inserted.InsertedID).
				Debug("inserted deal state into mongo")
			count += 1
		}
	}

	logger.With("count", count).Info("finished inserting deals into mongo")
	return nil
}
