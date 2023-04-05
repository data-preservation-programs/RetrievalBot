FROM public.ecr.aws/docker/library/golang:1.19
WORKDIR /app
COPY . .
RUN make build
# Main command to spin up workers to perform retrieval tasks
# CMD ["/app/RetrievalWorker.exe"]
# Start State Market Deals integration which will update the database with deals from the state market
# CMD ["/app/StateMarketDeals.exe"]
# Start Stub integration which pushes fake tasks to the queue
# CMD ["/app/StubIntegration.exe"]
