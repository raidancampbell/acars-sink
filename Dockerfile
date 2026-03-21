FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/acars-sink ./

FROM alpine:3.20

RUN adduser -D -g '' acars

COPY --from=builder /bin/acars-sink /usr/local/bin/acars-sink

WORKDIR /data
USER acars

EXPOSE 5555/tcp
EXPOSE 5555/udp
EXPOSE 5556/tcp

CMD ["/usr/local/bin/acars-sink"]
