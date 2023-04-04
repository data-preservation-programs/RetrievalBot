FROM public.ecr.aws/docker/library/golang:1.19
WORKDIR /app
COPY . .
RUN make build
# Main command to spin up workers to perform retrieval tasks
# CMD ["/app/RetrievalWorker.exe"]
# Start Stub integration which pushes fake tasks to the queue and will be handled by the workers
# CMD ["/app/StubIntegration.exe"]
