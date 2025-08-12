FROM golang:1.24.6@sha256:034848561f95a942e2163d9017e672f0c65403f699336db4529a908af00dfc98 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.6@sha256:034848561f95a942e2163d9017e672f0c65403f699336db4529a908af00dfc98 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
