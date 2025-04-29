FROM golang:1.24.2@sha256:8131d99bd9bdf38a3b9720c23c3d21f40d65137d4b2fd2eccab20e7ab7b5dee4 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.2@sha256:8131d99bd9bdf38a3b9720c23c3d21f40d65137d4b2fd2eccab20e7ab7b5dee4 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
