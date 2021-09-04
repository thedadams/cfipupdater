FROM golang:latest AS builder
COPY main.go .
RUN CGO_ENABLED=0 go build -o main main.go

FROM alpine

COPY --from=builder /go/main .

CMD ["/main"]
