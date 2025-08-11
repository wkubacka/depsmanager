FROM golang:1.24-bullseye AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

FROM debian:bullseye-slim
WORKDIR /app
ENV HTTP_PORT=8080
ENV DB_PATH=/data/storage.db

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates libc6 && rm -rf /var/lib/apt/lists/*

COPY --from=build /app/server /app/server

# Mount for SQLite file created by the app code
VOLUME ["/data"]

EXPOSE 8080
CMD ["/app/server"]
