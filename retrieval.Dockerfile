FROM alpine:latest
COPY ./retrieval_worker /bin/retrieval_worker
ENTRYPOINT  [ "/bin/retrieval_worker" ]
