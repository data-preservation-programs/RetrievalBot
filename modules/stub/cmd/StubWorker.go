package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/common"
	"github.com/data-preservation-programs/RetrievalBot/modules/stub"
)

func main() {
	worker := stub.Stub{}
	process, err := common.NewTaskWorkerProcess(context.Background(), common.Stub, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		panic(err)
	}
}
