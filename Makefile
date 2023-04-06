build:
	go build -o RetrievalWorker.exe ./common/cmd/RetrievalWorker
	go build -o StubWorker.exe ./modules/stub/cmd
	go build -o GraphsyncWorker.exe ./modules/graphsync/cmd
	go build -o HttpWorker.exe ./modules/http/cmd
	go build -o BitswapWorker.exe ./modules/bitswap/cmd
	go build -o StubIntegration.exe ./integrations/stub
	go build -o StateMarketDeals.exe ./integrations/statemarketdeals
	go build -o FilPlusIntegration.exe ./integrations/filplus

lint:
	cd common && golangci-lint run
	cd modules/stub && golangci-lint run
	cd modules/http && golangci-lint run
	cd modules/graphsync && golangci-lint run
	cd modules/bitswap && golangci-lint run
	cd integrations/stub && golangci-lint run
	cd integrations/statemarketdeals && golangci-lint run
	cd integrations/filplus && golangci-lint run
