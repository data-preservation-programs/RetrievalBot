package main

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/common"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	processManager, err := common.NewProcessManager()
	if err != nil {
		panic(err)
	}

	processManager.Run(context.Background())
}
