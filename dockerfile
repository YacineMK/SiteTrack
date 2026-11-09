FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sitetrack ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app

COPY --from=build /out/sitetrack /usr/local/bin/sitetrack

USER app
EXPOSE 8080
ENTRYPOINT ["sitetrack"]
