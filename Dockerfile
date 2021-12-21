# Builder
FROM golang:1 as builder
ARG http_proxy=""
ARG https_proxy=""
ARG no_proxy=""

WORKDIR /secretSync
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY / .
RUN CGO_ENABLED=0 GO111MODULE=on GOOS=linux go build -a -installsuffix cgo -o secrets-sync .

FROM alpine:latest

RUN mkdir -p /home/app && \
    addgroup app && \
    adduser -D -G app app

WORKDIR /home/app
COPY --from=builder /secretSync/secrets-sync .
RUN chown -R app:app /home/app
USER app
ENTRYPOINT ["./secrets-sync"]
