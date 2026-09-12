FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/identity ./services/identity/cmd/identity

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/identity /identity
EXPOSE 8081
ENTRYPOINT ["/identity"]
