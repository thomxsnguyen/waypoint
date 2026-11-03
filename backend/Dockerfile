FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /api ./cmd/api
FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 waypoint
USER waypoint
COPY --from=build /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
