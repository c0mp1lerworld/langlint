# syntax=docker/dockerfile:1
#
# Imagen del API de LangLint (contexto = apps/backend).
# Multi-stage: build estático en golang:1.26, runtime distroless nonroot (AP-MR9).
# Las migraciones goose van embebidas en el binario (//go:embed), no se copian.

# --- build ---
FROM golang:1.26 AS build
WORKDIR /src

# Cache de módulos: copiar solo los manifiestos primero.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/provisioner ./cmd/provisioner \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# --- runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/ /app/
EXPOSE 8080
USER nonroot:nonroot
# El entrypoint por defecto es el API; los jobs batch se invocan con:
#   docker run --entrypoint /app/provisioner <image> refresh-aggregates
ENTRYPOINT ["/app/api"]
