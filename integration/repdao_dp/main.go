package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DailyStats struct {
	ProviderID                string  `bson:"provider_id"`
	Date                      string  `bson:"date"`
	HTTPRetrievals            int     `bson:"http_retrievals"`
	HTTPRetrievalSuccess      int     `bson:"http_retrieval_success"`
	GraphSyncRetrievals       int     `bson:"graphsync_retrievals"`
	GraphSyncRetrievalSuccess int     `bson:"graphsync_retrieval_success"`
	BitswapRetrievals         int     `bson:"bitswap_retrievals"`
	BitswapRetrievalSuccess   int     `bson:"bitswap_retrieval_success"`
	AvgTTFBMS                 float64 `bson:"avg_ttfb_ms"`
	AvgSpeedBPS               float64 `bson:"avg_speed_bps"`
}

func main() {
	ctx := context.Background()
	retbotMongo, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("RESULT_MONGO_URI")))
	if err != nil {
		panic(err)
	}
	defer retbotMongo.Disconnect(ctx)

	retbot := retbotMongo.Database(os.Getenv("RESULT_MONGO_DATABASE")).Collection("task_result")
	repdaoMongo, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("REPDAO_MONGO_URI")))
	if err != nil {
		panic(err)
	}
	defer repdaoMongo.Disconnect(ctx)

	country := os.Getenv("RETRIEVER_COUNTRY")
	collectionName := os.Getenv("REPDAO_MONGO_COLLECTION")

	repdao := repdaoMongo.Database(os.Getenv("REPDAO_MONGO_DATABASE")).Collection(collectionName)

	// Find the last saved date
	var lastStats DailyStats
	err = repdao.FindOne(ctx, bson.D{}, options.FindOne().SetSort(bson.D{{"date", -1}})).Decode(&lastStats)
	if err != nil && err != mongo.ErrNoDocuments {
		panic(err)
	}

	var startDate time.Time
	if err == mongo.ErrNoDocuments {
		startDate = time.Time{}
	} else {
		startDate, err = time.Parse("2006-01-02", lastStats.Date)
		if err != nil {
			panic(err)
		}
		startDate = startDate.AddDate(0, 0, 1)
	}

	// Get the current day part of yesterday
	now := time.Now().UTC()
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	fmt.Printf("startDate: %s, endDate: %s\n", startDate, endDate)
	if startDate.After(endDate) || startDate.Equal(endDate) {
		fmt.Println("No new data to process")
		return
	}

	// Aggregate the results
	matchStage := bson.D{{"$match", bson.D{
		{"retriever.country", country},
		{"created_at", bson.D{{"$gte", startDate}, {"$lt", endDate}}},
		{"task.module", bson.D{{"$in", bson.A{"http", "bitswap", "graphsync"}}}},
	}}}

	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"provider_id", "$task.provider.id"},
			{"date", bson.D{{"$dateToString", bson.D{
				{"format", "%Y-%m-%d"},
				{"date", "$created_at"},
			}}}},
			{"module", "$task.module"},
			{"success", "$result.success"},
		}},
		{"count", bson.D{{"$sum", 1}}},
		{"ttfb_sum", bson.D{{"$sum", "$result.ttfb"}}},
		{"speed_sum", bson.D{{"$sum", "$result.speed"}}},
	}}}

	groupStage2 := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"provider_id", "$_id.provider_id"},
			{"date", "$_id.date"},
		}},
		{"http_retrievals", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$eq", bson.A{"$_id.module", "http"}}},
				"$count",
				0,
			}},
		}}}},
		{"http_retrieval_success", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$and", bson.A{
					bson.D{{"$eq", bson.A{"$_id.module", "http"}}},
					bson.D{{"$eq", bson.A{"$_id.success", true}}},
				}}},
				"$count",
				0,
			}}},
		}}},
		{"bitswap_retrievals", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$eq", bson.A{"$_id.module", "bitswap"}}},
				"$count",
				0,
			}},
		}}}},
		{"bitswap_retrieval_success", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$and", bson.A{
					bson.D{{"$eq", bson.A{"$_id.module", "bitswap"}}},
					bson.D{{"$eq", bson.A{"$_id.success", true}}},
				}}},
				"$count",
				0,
			}}},
		}}},
		{"graphsync_retrievals", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$eq", bson.A{"$_id.module", "graphsync"}}},
				"$count",
				0,
			}},
		}}}},
		{"graphsync_retrieval_success", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$and", bson.A{
					bson.D{{"$eq", bson.A{"$_id.module", "graphsync"}}},
					bson.D{{"$eq", bson.A{"$_id.success", true}}},
				}}},
				"$count",
				0,
			}}},
		}}},
		{"ttfb_sum", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$eq", bson.A{"$_id.success", true}}},
				"$ttfb_sum",
				0,
			}},
		}}}},
		{"speed_sum", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$eq", bson.A{"$_id.success", true}}},
				"$speed_sum",
				0,
			}},
		}}}},
		{"success_count", bson.D{{"$sum", bson.D{
			{"$cond", bson.A{
				bson.D{{"$eq", bson.A{"$_id.success", true}}},
				"$count",
				0,
			}},
		}}}},
	}}}

	projectStage := bson.D{{"$project", bson.D{
		{"provider_id", "$_id.provider_id"},
		{"date", "$_id.date"},
		{"http_retrievals", 1},
		{"http_retrieval_success", 1},
		{"bitswap_retrievals", 1},
		{"bitswap_retrieval_success", 1},
		{"graphsync_retrievals", 1},
		{"graphsync_retrieval_success", 1},
		{"avg_ttfb_ms", bson.D{{"$cond", bson.A{
			bson.D{{"$eq", bson.A{"$success_count", 0}}},
			0,
			bson.D{{"$divide", bson.A{bson.D{{"$divide", bson.A{"$ttfb_sum", "$success_count"}}}, 1000000.0}}},
		}}}},
		{"avg_speed_bps", bson.D{{"$cond", bson.A{
			bson.D{{"$eq", bson.A{"$success_count", 0}}},
			0,
			bson.D{{"$divide", bson.A{"$speed_sum", "$success_count"}}},
		}}}},
	}}}

	cursor, err := retbot.Aggregate(context.Background(), mongo.Pipeline{matchStage, groupStage, groupStage2, projectStage})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(context.Background())
	var stats []interface{}

	// Insert the aggregated results into the new collection
	for cursor.Next(context.Background()) {
		var dailyStats DailyStats
		err := cursor.Decode(&dailyStats)
		if err != nil {
			panic(err)
		}
		stats = append(stats, dailyStats)
		fmt.Printf("Got daily stats: %+v\n", dailyStats)
	}

	// Insert the aggregated results into the new collection
	result, err := repdao.InsertMany(ctx, stats)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Inserted %v documents into the new collection!\n", len(result.InsertedIDs))

}
