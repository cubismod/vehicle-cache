FROM golang:1.24.5@sha256:dd26b024adcdee20239479d884829d471fd821099b2f596f8f5d20b81bebca95 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.5@sha256:dd26b024adcdee20239479d884829d471fd821099b2f596f8f5d20b81bebca95 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
