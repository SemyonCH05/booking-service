FROM golang:1.26-alpine3.23 AS modules

COPY go.mod go.sum /modules/

WORKDIR /modules
RUN apk add --no-cache ca-certificates git

# RUN chmod 600 /root/.netrc
RUN go mod download

FROM golang:1.26-alpine3.23 AS builder

COPY --from=modules /go/pkg /go/pkg
COPY . /app

WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s" -tags migrate -o /bin/app ./cmd/app

FROM scratch 

COPY --from=builder /app/config /config
COPY --from=builder /app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app"]
