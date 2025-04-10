FROM golang:1.24.2@sha256:18a1f2d1e1d3c49f27c904e9182375169615c65852ace724987929b910195b2c AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.2@sha256:18a1f2d1e1d3c49f27c904e9182375169615c65852ace724987929b910195b2c AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
