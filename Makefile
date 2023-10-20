build:
	go1.20.1 build -o retrieval_worker ./pkg/cmd/retrieval_worker
	go1.20.1 build -o stub_worker ./worker/stub/cmd
	go1.20.1 build -o graphsync_worker ./worker/graphsync/cmd
	go1.20.1 build -o http_worker ./worker/http/cmd
	go1.20.1 build -o bitswap_worker ./worker/bitswap/cmd
	go1.20.1 build -o oneoff_integration ./integration/oneoff
	go1.20.1 build -o statemarketdeals ./integration/statemarketdeals
	go1.20.1 build -o filplus_integration ./integration/filplus
	go1.20.1 build -o repdao ./integration/repdao
	go1.20.1 build -o spadev0 ./integration/spadev0
	go1.20.1 build -o repdao_dp ./integration/repdao_dp
	go1.20.1 build -o spcoverage ./integration/spcoverage

lint:
	gofmt -s -w .
	golangci-lint run --fix --timeout 10m
	staticcheck ./...
