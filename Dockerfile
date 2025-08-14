FROM golang:1.24.6@sha256:6226a5cbe59f576b8a93f381d4f818e170001874c90b8c06a6cacac8616ba3aa AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.6@sha256:6226a5cbe59f576b8a93f381d4f818e170001874c90b8c06a6cacac8616ba3aa AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
