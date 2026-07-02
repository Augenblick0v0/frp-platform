FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY client/frp-client/go.mod ./client/frp-client/go.mod
COPY client/frp-client ./client/frp-client
WORKDIR /src/client/frp-client
RUN go build -o /out/frp-client .

FROM alpine:3.20
WORKDIR /opt/frp-client
COPY --from=builder /out/frp-client /opt/frp-client/frp-client
COPY apps/client-webui /opt/frp-client/webui
EXPOSE 18080
CMD ["/opt/frp-client/frp-client", "-addr", "0.0.0.0:18080", "-web", "/opt/frp-client/webui"]
