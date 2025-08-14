FROM golang:1.24.6@sha256:61808652990bcaa6981db6a85ecd0099c8fa10a6d49c3bd40194c00b69917856 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.6@sha256:61808652990bcaa6981db6a85ecd0099c8fa10a6d49c3bd40194c00b69917856 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
