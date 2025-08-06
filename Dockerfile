FROM golang:1.24.6@sha256:2c89c41fb9efc3807029b59af69645867cfe978d2b877d475be0d72f6c6ce6f6 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.6@sha256:2c89c41fb9efc3807029b59af69645867cfe978d2b877d475be0d72f6c6ce6f6 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
