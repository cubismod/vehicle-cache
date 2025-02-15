FROM golang:1.24.0 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.0 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
