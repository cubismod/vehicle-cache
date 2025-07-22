FROM golang:1.24.5@sha256:fdcd2e5a34587bd5426c90e1531fd5ba448c89bb738df0f33860dfc69439a1f5 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.5@sha256:fdcd2e5a34587bd5426c90e1531fd5ba448c89bb738df0f33860dfc69439a1f5 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
