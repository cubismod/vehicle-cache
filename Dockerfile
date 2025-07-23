FROM golang:1.24.5@sha256:a3bb6cd5f068b34961d60dcd7fc51fb70df7b74b4aa89ac480fc38a6ccba265e AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.5@sha256:a3bb6cd5f068b34961d60dcd7fc51fb70df7b74b4aa89ac480fc38a6ccba265e AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
