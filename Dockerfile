FROM golang:1.24.2@sha256:c0b66cfec8562c8d8452c63a63ede2014526224fa1cd5ecf5acda73f8b28263b AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.2@sha256:c0b66cfec8562c8d8452c63a63ede2014526224fa1cd5ecf5acda73f8b28263b AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
