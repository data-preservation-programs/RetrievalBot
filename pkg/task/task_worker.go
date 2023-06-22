package task

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

type Worker interface {
	DoWork(task Task) (*RetrievalResult, error)
}

type WorkerProcess struct {
	id                 uuid.UUID
	taskCollection     *mongo.Collection
	resultCollection   *mongo.Collection
	worker             Worker
	module             ModuleName
	acceptedContinents string
	acceptedCountries  string
	pollInterval       time.Duration
	retrieverInfo      Retriever
	timeoutBuffer      time.Duration
}

func (t WorkerProcess) Close() {
	// nolint:errcheck
	t.taskCollection.Database().Client().Disconnect(context.Background())
	// nolint:errcheck
	t.resultCollection.Database().Client().Disconnect(context.Background())
}

func NewTaskWorkerProcess(
	ctx context.Context,
	module ModuleName,
	worker Worker) (*WorkerProcess, error) {
	taskClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.QueueMongoURI)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo queueDB")
	}

	taskCollection := taskClient.Database(env.GetRequiredString(env.QueueMongoDatabase)).Collection("task_queue")

	resultClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env.GetRequiredString(env.ResultMongoURI)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongo resultDB")
	}

	resultCollection := resultClient.Database(env.GetRequiredString(env.ResultMongoDatabase)).Collection("task_result")

	retrieverInfo := Retriever{
		PublicIP:  env.GetRequiredString(env.PublicIP),
		City:      env.GetRequiredString(env.City),
		Region:    env.GetRequiredString(env.Region),
		Country:   env.GetRequiredString(env.Country),
		Continent: env.GetRequiredString(env.Continent),
		ASN:       env.GetRequiredString(env.ASN),
		ISP:       env.GetRequiredString(env.ISP),
		Latitude:  env.GetRequiredFloat32(env.Latitude),
		Longitude: env.GetRequiredFloat32(env.Longitude),
	}

	id := uuid.New()

	return &WorkerProcess{
		id,
		taskCollection,
		resultCollection,
		worker,
		module,
		env.GetString(env.AcceptedContinents, ""),
		env.GetString(env.AcceptedCountries, ""),
		env.GetDuration(env.TaskWorkerPollInterval, 10*time.Second),
		retrieverInfo,
		env.GetDuration(env.TaskWorkerTimeoutBuffer, 10*time.Second),
	}, nil
}

func (t WorkerProcess) Poll(ctx context.Context) error {
	logger := logging.Logger("task-worker").With("protocol", t.module, "workerId", t.id)
	var singleResult *mongo.SingleResult
	for {
		logger.Debug("polling for task")

		//nolint:govet
		match := bson.D{
			{"module", t.module},
		}

		if len(t.acceptedCountries) > 0 {
			if strings.HasPrefix(t.acceptedCountries, "!") {
				match = append(match, bson.E{Key: "provider.country",
					Value: bson.D{{Key: "$nin", Value: strings.Split(t.acceptedCountries[1:], ",")}}})
			} else {
				match = append(match, bson.E{Key: "provider.country",
					Value: bson.D{{Key: "$in", Value: strings.Split(t.acceptedCountries, ",")}}})
			}
		}

		if len(t.acceptedContinents) > 0 {
			if strings.HasPrefix(t.acceptedCountries, "!") {
				match = append(match, bson.E{Key: "provider.continent",
					Value: bson.D{{Key: "$nin", Value: strings.Split(t.acceptedContinents[1:], ",")}}})
			} else {
				match = append(match, bson.E{Key: "provider.continent",
					Value: bson.D{{Key: "$in", Value: strings.Split(t.acceptedContinents, ",")}}})
			}
		}

		logger.With("filter", match).Debug("FindOneAndDelete")
		singleResult = t.taskCollection.FindOneAndDelete(ctx, match,
			options.FindOneAndDelete().SetSort(bson.D{{Key: "created_at", Value: 1}}))
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
	resultChan := make(chan RetrievalResult)
	errChan := make(chan error)
	go func() {
		result, err := t.worker.DoWork(*found)
		if err != nil {
			errResult := resolveErrorResult(err)
			if errResult != nil {
				resultChan <- *errResult
			} else {
				logger.With("error", err).Error("failed to do work")
				errChan <- err
			}
		} else {
			resultChan <- *result
		}
	}()

	var retrievalResult RetrievalResult
	select {
	case <-ctx.Done():
		//nolint:wrapcheck
		return ctx.Err()
	case <-time.After(found.Timeout + t.timeoutBuffer):
		retrievalResult = *NewErrorRetrievalResult(Timeout, errors.Errorf("timed out after %s", found.Timeout))
	case r := <-resultChan:
		retrievalResult = r
	case err = <-errChan:
		return err
	}

	taskResult := Result{
		Task:      *found,
		Result:    retrievalResult,
		Retriever: t.retrieverInfo,
		CreatedAt: time.Now().UTC(),
	}

	insertResult, err := t.resultCollection.InsertOne(ctx, taskResult)
	if err != nil {
		return errors.Wrap(err, "failed to insert result")
	}

	logger.With("result", retrievalResult, "InsertedID", insertResult.InsertedID).Info("inserted result")
	return nil
}
