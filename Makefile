build:
	go build -o RetrievalWorker.exe ./pkg/cmd/retrieval_worker
	go build -o StubWorker.exe ./worker/stub/cmd
	go build -o GraphsyncWorker.exe ./worker/graphsync/cmd
	go build -o HttpWorker.exe ./worker/http/cmd
	go build -o BitswapWorker.exe ./worker/bitswap/cmd
	go build -o OneOffIntegration.exe ./integration/oneoff
	go build -o StateMarketDeals.exe ./integration/statemarketdeals
	go build -o FilPlusIntegration.exe ./integration/filplus

lint:
	gofmt -s -w .
	cd pkg && golangci-lint run
	cd worker/stub && golangci-lint run
	cd worker/http && golangci-lint run
	cd worker/graphsync && golangci-lint run
	cd worker/bitswap && golangci-lint run
	cd integration/oneoff && golangci-lint run
	cd integration/statemarketdeals && golangci-lint run
	cd integration/filplus && golangci-lint run
