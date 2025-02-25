FROM golang:1.24.0@sha256:5255fad61a7e8880e742ee3e30ac54d3fdc48ea5236d0bcf14bfedb6643cbeae AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.0@sha256:5255fad61a7e8880e742ee3e30ac54d3fdc48ea5236d0bcf14bfedb6643cbeae AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
