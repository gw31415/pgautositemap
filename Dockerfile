# Build
FROM golang:1.25.1 as builder
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o app

# Final minimum image
FROM gcr.io/distroless/static-debian12
COPY --from=builder /src/app /app
CMD ["/app"]
