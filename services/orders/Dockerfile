FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/orders ./services/orders/cmd/orders

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/orders /orders
EXPOSE 8083
ENTRYPOINT ["/orders"]
