FROM golang:1.25.0@sha256:9e56f0d0f043a68bb8c47c819e47dc29f6e8f5129b8885bed9d43f058f7f3ed6 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.25.0@sha256:9e56f0d0f043a68bb8c47c819e47dc29f6e8f5129b8885bed9d43f058f7f3ed6 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
