FROM golang:alpine as builder

WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 go build


FROM alpine

WORKDIR /app
RUN mkdir -p /root/.config/timoneiro && chmod 700 /root/.config/timoneiro
COPY --from=builder /app/timoneiro .
COPY --from=builder /app/timoneiro_init-sample.filo /root/.config/timoneiro/init.filo

CMD ["/app/timoneiro"]
