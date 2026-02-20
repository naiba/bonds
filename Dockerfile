FROM golang:1.25-alpine AS swagger

RUN apk add --no-cache gcc musl-dev sqlite-dev
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /build
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

FROM oven/bun:1.3-alpine AS frontend
ARG VERSION=dev
ENV VITE_APP_VERSION=${VERSION}

WORKDIR /build/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen
COPY web/ .
COPY --from=swagger /build/docs/swagger.json /build/server/docs/swagger.json
RUN bun run gen:api && bun run build

FROM golang:1.25-alpine AS backend
ARG VERSION=dev

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
COPY --from=swagger /build/docs ./docs/
COPY --from=frontend /build/web/dist ./internal/frontend/dist/

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o bonds-server cmd/server/main.go

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
