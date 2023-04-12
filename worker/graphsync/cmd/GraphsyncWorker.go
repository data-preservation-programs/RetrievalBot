package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/data-preservation-programs/RetrievalBot/worker/graphsync"
	logging "github.com/ipfs/go-log/v2"
)

func main() {
	worker := graphsync.Worker{}
	process, err := task.NewTaskWorkerProcess(context.Background(), task.GraphSync, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		logging.Logger("task-worker").With("protocol", task.HTTP).Error(err)
	}
}
