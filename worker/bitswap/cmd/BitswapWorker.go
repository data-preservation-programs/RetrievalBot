package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/data-preservation-programs/RetrievalBot/worker/bitswap"
)

func main() {
	worker := bitswap.Worker{}
	process, err := task.NewTaskWorkerProcess(context.Background(), task.Bitswap, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		panic(err)
	}
}
