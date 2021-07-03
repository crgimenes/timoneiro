FROM golang:alpine as builder

WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 go build


FROM alpine

WORKDIR /app
COPY --from=builder /app/timoneiro .

CMD ["/app/timoneiro"]