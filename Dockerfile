FROM golang AS builder
RUN go version
COPY . /go/src/github.com/46bit/octopus-clickhouse
WORKDIR /go/src/github.com/46bit/octopus-clickhouse
RUN CGO_ENABLED=0 \
    go build \
    -trimpath \
    -o octopus-clickhouse \
    .

FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/github.com/46bit/octopus-clickhouse/octopus-clickhouse /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/octopus-clickhouse"]
