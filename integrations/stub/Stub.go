package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/common"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

func main() {
	logger := logging.Logger("stub-integration")
	ctx := context.Background()
	taskClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("QUEUE_MONGO_URI")))
	if err != nil {
		panic(err)
	}
	taskCollection := taskClient.Database(os.Getenv("QUEUE_MONGO_DATABASE")).Collection("task_queue")

	for {
		task := common.Task{
			Requester: "stub",
			Metadata:  nil,
			Provider: common.Provider{
				ID:        "FakeID",
				Country:   "US",
				Continent: "NA",
			},
			Protocol: common.Protocol{
				Name: common.Stub,
			},
			Content: common.Content{
				CID: "FakeCID",
			},
			Strategy: common.Strategy{
				Type: "stub",
			},
			CreatedAt: time.Now().UTC(),
		}
		insertResult, err := taskCollection.InsertOne(ctx, task)
		if err != nil {
			logger.Errorf("failed to insert task: %s", err)
		} else {
			logger.Infof("inserted task with id: %s", insertResult.InsertedID)
		}

		time.Sleep(time.Minute)
	}
}