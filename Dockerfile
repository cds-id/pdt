# Stage 1: Build frontend
FROM oven/bun:alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package.json ./
RUN bun install
COPY frontend/ ./
RUN bun run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend-build
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /server cmd/server/main.go

# Stage 3: Production
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=backend-build /server .
COPY --from=frontend-build /app/frontend/dist ./public

EXPOSE 8080

CMD ["./server"]
