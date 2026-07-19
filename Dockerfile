FROM golang:1.24-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod ./
RUN go mod download
COPY backend/ .
RUN mkdir -p /app/bin \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/migrate ./cmd/migrate

FROM alpine:3.21 AS backend-runtime
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-builder /app/bin/api /app/api
COPY --from=backend-builder /app/bin/migrate /app/migrate
EXPOSE 8080
CMD ["/app/api"]

FROM node:22-alpine AS frontend-development
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
