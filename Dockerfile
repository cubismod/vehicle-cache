FROM golang:1.24.4@sha256:270cd5365c84dd24716c42d7f9f7ddfbc131c8687e163e6748b9c1322c518213 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.4@sha256:270cd5365c84dd24716c42d7f9f7ddfbc131c8687e163e6748b9c1322c518213 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
