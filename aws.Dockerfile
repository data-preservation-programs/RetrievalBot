FROM public.ecr.aws/docker/library/golang:1.19-alpine as builder
WORKDIR /app
COPY . .
RUN go build -o retrieval_worker ./pkg/cmd/retrieval_worker \
    && go build -o graphsync_worker ./worker/graphsync/cmd \
    && go build -o http_worker ./worker/http/cmd \
    && go build -o bitswap_worker ./worker/bitswap/cmd

FROM public.ecr.aws/docker/library/alpine:latest
WORKDIR /app
COPY --from=builder /app/retrieval_worker .
COPY --from=builder /app/graphsync_worker .
COPY --from=builder /app/http_worker .
COPY --from=builder /app/bitswap_worker .
CMD ["/app/retrieval_worker"]
