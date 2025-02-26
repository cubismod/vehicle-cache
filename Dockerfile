FROM golang:1.24.0@sha256:58cf31c1b10858d5772475e4f86853a03e0e220cc6cfd60965053951a2563b5e AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.0@sha256:58cf31c1b10858d5772475e4f86853a03e0e220cc6cfd60965053951a2563b5e AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
