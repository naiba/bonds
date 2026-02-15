FROM oven/bun:1.3-alpine AS frontend

WORKDIR /build
COPY web/package.json web/bun.lock ./
RUN bun install --frozen
COPY web/ .
RUN bun run build

FROM golang:1.25-alpine AS backend

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build
COPY server/go.mod server/go.sum ./
RUN go mod download
RUN go install github.com/swaggo/swag/cmd/swag@latest
COPY server/ .
COPY --from=frontend /build/dist ./internal/frontend/dist/

RUN swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o bonds-server cmd/server/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates sqlite-libs tzdata
WORKDIR /app
COPY --from=backend /build/bonds-server .
RUN mkdir -p /app/data

EXPOSE 8080
ENV DB_DRIVER=sqlite \
    DB_DSN=/app/data/bonds.db \
    APP_ENV=production

CMD ["./bonds-server"]
