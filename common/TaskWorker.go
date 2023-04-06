package common

import (
	"context"
	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"strconv"
	"strings"
	"time"
)

type TaskWorker interface {
	DoWork(task Task) (RetrievalResult, error)
}

type TaskWorkerProcess struct {
	id                 uuid.UUID
	taskCollection     *mongo.Collection
	resultCollection   *mongo.Collection
	worker             TaskWorker
	module             ModuleName
	acceptedContinents []string
	acceptedCountries  []string
	pollInterval       time.Duration
	retrieverInfo      Retriever
}

func GetRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		logging.Logger("init").Panicf("%s not set", key)
	}

	return value
}

func (t TaskWorkerProcess) Close() {
	// nolint:errcheck
	t.taskCollection.Database().Client().Disconnect(context.Background())
	// nolint:errcheck
	t.resultCollection.Database().Client().Disconnect(context.Background())
}

func NewTaskWorkerProcess(
	ctx context.Context,
	module ModuleName,
	worker TaskWorker) (*TaskWorkerProcess, error) {
	taskClient, err := mongo.Connect(ctx, options.Client().ApplyURI(GetRequiredEnv("QUEUE_MONGO_URI")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo queueDB")
	}

	taskCollection := taskClient.Database(GetRequiredEnv("QUEUE_MONGO_DATABASE")).Collection("task_queue")

	resultClient, err := mongo.Connect(ctx, options.Client().ApplyURI(GetRequiredEnv("RESULT_MONGO_URI")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo resultDB")
	}

	resultCollection := resultClient.Database(GetRequiredEnv("RESULT_MONGO_DATABASE")).Collection("task_result")

	acceptedContinents := make([]string, 0)
	if os.Getenv("ACCEPTED_CONTINENTS") != "" {
		acceptedContinents = strings.Split(os.Getenv("ACCEPTED_CONTINENTS"), ",")
	}

	acceptedCountries := make([]string, 0)
	if os.Getenv("ACCEPTED_COUNTRIES") != "" {
		acceptedCountries = strings.Split(os.Getenv("ACCEPTED_COUNTRIES"), ",")
	}

	pollIntervalString := os.Getenv("POLL_INTERVAL_SECOND")
	if pollIntervalString == "" {
		pollIntervalString = "10"
	}

	pollIntervalNumber, err := strconv.Atoi(pollIntervalString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse poll interval")
	}

	pollInterval := time.Duration(pollIntervalNumber) * time.Second
	latitudeString := os.Getenv("_LATITUDE")
	longitudeString := os.Getenv("_LONGITUDE")
	latitude, err := strconv.ParseFloat(latitudeString, 32)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse latitude")
	}
	longitude, err := strconv.ParseFloat(longitudeString, 32)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse longitude")
	}
	retrieverInfo := Retriever{
		PublicIP:  GetRequiredEnv("_PUBLIC_IP"),
		City:      GetRequiredEnv("_CITY"),
		Region:    GetRequiredEnv("_REGION"),
		Country:   os.Getenv("_COUNTRY"),
		Continent: os.Getenv("_CONTINENT"),
		ASN:       GetRequiredEnv("_ASN"),
		Org:       GetRequiredEnv("_ORG"),
		Latitude:  float32(latitude),
		Longitude: float32(longitude),
	}

	id := uuid.New()

	return &TaskWorkerProcess{
		id,
		taskCollection,
		resultCollection,
		worker,
		module,
		acceptedContinents,
		acceptedCountries,
		pollInterval,
		retrieverInfo,
	}, nil
}

func (t TaskWorkerProcess) Poll(ctx context.Context) error {
	logger := logging.Logger("task-worker").With("protocol", t.module, "workerId", t.id)
	var singleResult *mongo.SingleResult
	for {
		logger.Debug("polling for task")

		//nolint:govet
		match := bson.D{
			{"module", t.module},
		}

		if len(t.acceptedCountries) > 0 {
			//nolint:govet
			match = append(match, bson.E{Key: "provider.country", Value: bson.D{{"$in", t.acceptedCountries}}})
		}

		if len(t.acceptedContinents) > 0 {
			//nolint:govet
			match = append(match, bson.E{Key: "provider.continent", Value: bson.D{{"$in", t.acceptedContinents}}})
		}

		logger.With("filter", match).Debug("FindOneAndDelete")
		singleResult = t.taskCollection.FindOneAndDelete(ctx, match)
		if errors.Is(singleResult.Err(), mongo.ErrNoDocuments) {
			logger.Debug("no task singleResult")
			time.Sleep(t.pollInterval)
			continue
		}

		if singleResult.Err() != nil {
			return errors.Wrap(singleResult.Err(), "failed to find task")
		}

		break
	}

	found := new(Task)
	err := singleResult.Decode(found)
	if err != nil {
		return errors.Wrap(err, "failed to decode task")
	}

	logger.With("task", found).Info("found new task")
	result, err := t.worker.DoWork(*found)
	if err != nil {
		return errors.Wrap(err, "failed to do work")
	}

	taskResult := TaskResult{
		Task:      *found,
		Result:    result,
		Retriever: t.retrieverInfo,
		CreatedAt: time.Now().UTC(),
	}

	insertResult, err := t.resultCollection.InsertOne(ctx, taskResult)
	if err != nil {
		return errors.Wrap(err, "failed to insert result")
	}

	logger.With("result", result, "InsertedID", insertResult.InsertedID).Info("inserted result")
	return nil
}
