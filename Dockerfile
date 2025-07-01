FROM golang:1.24.4@sha256:1aa97ddeb238eba47a930016f676e46f471e4e26e82d1a353ff2bf42304a48e2 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.4@sha256:1aa97ddeb238eba47a930016f676e46f471e4e26e82d1a353ff2bf42304a48e2 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
