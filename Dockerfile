FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/stats-service ./cmd/stats-service

FROM alpine:3.22

RUN adduser -D -H -u 10001 app
COPY --from=build /out/stats-service /usr/local/bin/stats-service

USER app
ENTRYPOINT ["/usr/local/bin/stats-service"]
