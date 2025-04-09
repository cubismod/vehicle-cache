FROM golang:1.24.2@sha256:227d106dca555769db9977f33e5d3d27422c5e75af1afc080b92f390c326de80 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 go build -buildvcs=false -o vehicle-cache-bin ./vehicle-cache

FROM golang:1.24.2@sha256:227d106dca555769db9977f33e5d3d27422c5e75af1afc080b92f390c326de80 AS prod

COPY --from=builder /build/vehicle-cache-bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vehicle-cache-bin"]
