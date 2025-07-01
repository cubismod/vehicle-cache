FROM golang:1.24.4@sha256:1bb140b73b0c33df854496d07cb8d05ac62b358923ab513c18ede977f880c8b6 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.4@sha256:1bb140b73b0c33df854496d07cb8d05ac62b358923ab513c18ede977f880c8b6 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
