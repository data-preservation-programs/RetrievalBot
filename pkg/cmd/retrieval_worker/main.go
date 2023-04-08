package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/process"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	processManager, err := process.NewProcessManager()
	if err != nil {
		panic(err)
	}

	processManager.Run(context.Background())
}
