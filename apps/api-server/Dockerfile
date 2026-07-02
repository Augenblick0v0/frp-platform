FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY apps/api-server/go.mod ./apps/api-server/go.mod
COPY apps/api-server ./apps/api-server
WORKDIR /src/apps/api-server
RUN go build -o /out/api-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache certbot bind-tools
WORKDIR /app
COPY --from=builder /out/api-server /app/api-server
ENV HTTP_ADDR=:8080
EXPOSE 8080
CMD ["/app/api-server"]
