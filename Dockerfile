# Estágio 1: Build do binário Go
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download 2>/dev/null || true

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /migrate ./cmd/migrate

# Estágio 2: Imagem mínima para execução
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /api .
COPY --from=builder /migrate .
COPY migrations ./migrations

EXPOSE 8080

# Aplica as migrações pendentes antes de subir a API; se falharem, a API não sobe
CMD ["sh", "-c", "./migrate && exec ./api"]
