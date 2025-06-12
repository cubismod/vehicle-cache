FROM golang:1.24.4@sha256:3178db8b0d0fbcb11cf8271f7fb75b5f1f76367e306968c7c725999cb30c9982 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.4@sha256:3178db8b0d0fbcb11cf8271f7fb75b5f1f76367e306968c7c725999cb30c9982 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
