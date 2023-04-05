build:
	go build -o RetrievalWorker.exe ./common/cmd/RetrievalWorker
	go build -o StubWorker.exe ./modules/stub/cmd
	go build -o StubIntegration.exe ./integrations/stub
	go build -o StateMarketDeals.exe ./integrations/statemarketdeals
	go build -o FilPlusIntegration.exe ./integrations/filplus

lint:
	cd common && golangci-lint run
	cd modules/stub && golangci-lint run
	cd integrations/stub && golangci-lint run
	cd integrations/statemarketdeals && golangci-lint run
	cd integrations/filplus && golangci-lint run
