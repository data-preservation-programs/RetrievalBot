build:
	go build -o RetrievalWorker.exe ./common/cmd/RetrievalWorker
	go build -o StubWorker.exe ./modules/stub/cmd
	go build -o StubIntegration.exe ./integrations/stub

lint:
	cd common && golangci-lint run
	cd modules/stub && golangci-lint run
	cd integrations/stub && golangci-lint run
