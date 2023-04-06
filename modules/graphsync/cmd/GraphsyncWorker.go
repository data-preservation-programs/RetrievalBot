package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/common"
	"github.com/data-preservation-programs/RetrievalBot/modules/graphsync"
)

func main() {
	worker := graphsync.Worker{}
	process, err := common.NewTaskWorkerProcess(context.Background(), common.GraphSync, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		panic(err)
	}
}
