FROM golang:1.22-alpine AS build
WORKDIR /src/apps/api-server
COPY apps/api-server/go.mod apps/api-server/go.sum ./
RUN go mod download
COPY apps/api-server ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/node-agent ./cmd/node-agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates certbot docker-cli
COPY --from=build /out/node-agent /usr/local/bin/node-agent
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/node-agent"]
