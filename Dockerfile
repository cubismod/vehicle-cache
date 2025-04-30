FROM golang:1.24.2@sha256:3a060d683c28fbb21d7fe8966458e084a6d7ebfb1f3ef3fd901abd2083c43675 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.2@sha256:3a060d683c28fbb21d7fe8966458e084a6d7ebfb1f3ef3fd901abd2083c43675 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
