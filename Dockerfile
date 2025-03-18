FROM golang:1.24.1@sha256:ceb568d0de81fbef40ce4fee77eab524a0a0a8536065c51866ad8c59b7a912cf AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.1@sha256:ceb568d0de81fbef40ce4fee77eab524a0a0a8536065c51866ad8c59b7a912cf AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
