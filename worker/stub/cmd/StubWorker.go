package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/data-preservation-programs/RetrievalBot/worker/stub"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	worker := stub.Worker{}
	process, err := task.NewTaskWorkerProcess(context.Background(), task.Stub, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		panic(err)
	}
}
