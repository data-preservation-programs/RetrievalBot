package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/data-preservation-programs/RetrievalBot/worker/http"
)

func main() {
	worker := http.Worker{}
	process, err := task.NewTaskWorkerProcess(context.Background(), task.HTTP, worker)
	if err != nil {
		panic(err)
	}

	defer process.Close()

	err = process.Poll(context.Background())
	if err != nil {
		panic(err)
	}
}
