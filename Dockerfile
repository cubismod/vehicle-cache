FROM golang:1.24.2@sha256:b6652731d1d5622f85509f72942ce2a344e3bf6dd6793b2e462cc5fb3126b566 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.2@sha256:b6652731d1d5622f85509f72942ce2a344e3bf6dd6793b2e462cc5fb3126b566 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
