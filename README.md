# RetrievalBot

The goal of retrieval bot is to offer a scalable framework to perform retrieval testing over Filecoin network. 

There is no centralized orchestrator to manage retrieval queue or work. Instead, it uses MongoDB to manage work queue as well as saving retrieval results.

## Workers
Workers refer to the unit that consumes the worker queue. There are 4 basic types of workers as of now.

### Stub Worker
This type of worker does nothing but saves random result to the database. It is used to test the database connection and the queue.

### Bitswap Worker
This worker currently only support retrieving a single block from the storage provider:
1. Lookup the provider's libp2p protocols
2. If it is using boost market, then lookup the supported retrieval protocols
3. Find the bitswap protocol info and make a single block retrieval

### Graphsync Worker
This worker currently only support retrieving the root block from the storage provider:
1. Make graphsync retrieval with selector that only matches root block from the storage provider

### HTTP Worker
This worker currently only support retrieving the first few MiB of the pieces from the storage provider:
1. Lookup the provider's libp2p protocols
2. If it is using boost market, then lookup the supported retrieval protocols
3. Find the HTTP protocol info and make the retrieval for up to first few MiB

## Integrations
Integrations refer to the unit that either pushes work item to the retrieval queue, or other long-running jobs that may interact with the database in different ways

### StateMarketDeals Integration
This integration periodically pulls the statemarketdeals.json from GLIP API and saves it to the database.

### FILPLUS Integration
This integration pulls random active deals from StateMarketDeals database and push Bitswap/Graphsync/HTTP retrieval workitems into the work queue.

### Oneoff Integration
This integration push a single workitem into the queue with command line arguments, i.e.
```shell
./OneOffIntegration.exe http f0xxxx baxxxx
```

## Get started
1. Setup a mongodb server
2. Run the software natively or via a docker with environment variables. Please see [.env.retrievalworker](./.env.retrievalworker), [.env.statemarketdeals](./.env.statemarketdeals), [.env.filplus](./.env.filplus),  to find all relevant keys required. 
* Do not run `StubWorker.exe`, `GraphsyncWorker.exe`, `HttpWorker.exe`, `BitswapWorker.exe` directly as they are invoked by `RetrievalWorker.exe` to offer sandboxed environment.
* For a mininal Filplus validation, `RetrievalWorker.exe`, `StateMarketDeals.exe` and `FilPlusIntegration.exe` needs to be run.
