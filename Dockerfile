FROM golang:1.26.11-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api .


FROM node:22-alpine AS frontend-development
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]


FROM alpine:3.21 AS backend-runtime
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-builder /app/api /app/api
EXPOSE 8080
CMD ["/app/api"]
