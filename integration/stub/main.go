package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/joho/godotenv/autoload"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var logger = logging.Logger("stub-integration")

func main() {
	ctx := context.Background()
	taskClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString("QUEUE_MONGO_URI")))
	if err != nil {
		panic(err)
	}
	taskCollection := taskClient.Database(env.GetRequiredString("QUEUE_MONGO_DATABASE")).Collection("task_queue")

	for {
		tsk := task.Task{
			Requester: "stub",
			Module:    task.Stub,
			Metadata:  nil,
			Provider: task.Provider{
				ID:         "FakeID",
				PeerID:     "FakePeerID",
				Multiaddrs: nil,
				City:       "New York",
				Region:     "NY",
				Country:    "US",
				Continent:  "NA",
			},
			Content: task.Content{
				CID: "FakeCID",
			},
			CreatedAt: time.Now().UTC(),
			Timeout:   time.Minute,
		}
		insertResult, err := taskCollection.InsertOne(ctx, tsk)
		if err != nil {
			logger.Errorf("failed to insert task: %s", err)
		} else {
			logger.Infof("inserted task with id: %s", insertResult.InsertedID)
		}

		time.Sleep(time.Minute)
	}
}
