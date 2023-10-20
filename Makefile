build:
	go build -o retrieval_worker ./pkg/cmd/retrieval_worker
	go build -o stub_worker ./worker/stub/cmd
	go build -o graphsync_worker ./worker/graphsync/cmd
	go build -o http_worker ./worker/http/cmd
	go build -o bitswap_worker ./worker/bitswap/cmd
	go build -o oneoff_integration ./integration/oneoff
	go build -o statemarketdeals ./integration/statemarketdeals
	go build -o filplus_integration ./integration/filplus
	go build -o repdao ./integration/repdao
	go build -o spadev0 ./integration/spadev0
	go build -o repdao_dp ./integration/repdao_dp
	go build -o spcoverage ./integration/spcoverage

lint:
	gofmt -s -w .
	golangci-lint run --fix --timeout 10m
	staticcheck ./...
