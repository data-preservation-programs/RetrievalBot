package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/common"
	"github.com/data-preservation-programs/RetrievalBot/modules/bitswap"
)

func main() {
	worker := bitswap.Worker{}
	process, err := common.NewTaskWorkerProcess(context.Background(), common.Bitswap, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		panic(err)
	}
}
