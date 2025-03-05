FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.mod ./
COPY *.go ./

RUN go mod download
RUN CGO_ENABLED=0 go build -o /ghttpd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /ghttpd /ghttpd
ENTRYPOINT ["/ghttpd"]
