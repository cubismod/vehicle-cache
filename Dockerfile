FROM golang:1.24.6@sha256:a18e9e0d94dcc4fffb5c6fa5ec8580edd2e8adcf6541e53304f2bfcaafafd52e AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.6@sha256:a18e9e0d94dcc4fffb5c6fa5ec8580edd2e8adcf6541e53304f2bfcaafafd52e AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
