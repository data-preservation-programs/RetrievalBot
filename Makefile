build:
	go build -o retrieval_worker ./pkg/cmd/retrieval_worker
	go build -o stub_worker ./worker/stub/cmd
	go build -o graphsync_worker ./worker/graphsync/cmd
	go build -o http_worker ./worker/http/cmd
	go build -o bitswap_worker ./worker/bitswap/cmd
	go build -o oneoff_integration ./integration/oneoff
	go build -o statemarketdeals ./integration/statemarketdeals
	go build -o filplus_integration ./integration/filplus

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
