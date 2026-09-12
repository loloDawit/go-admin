FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway ./services/gateway/cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gateway /gateway
EXPOSE 8080
# Distroless: no shell, curl, or wget, so the health check runs the binary
# itself against its own /healthz rather than an external HTTP client.
HEALTHCHECK --interval=3s --timeout=2s --start-period=10s --retries=5 \
  CMD ["/gateway", "-health-check"]
ENTRYPOINT ["/gateway"]
