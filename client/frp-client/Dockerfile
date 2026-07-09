FROM golang:1.22-alpine AS go-builder
WORKDIR /src
COPY client/frp-client/go.mod ./client/frp-client/go.mod
COPY client/frp-client ./client/frp-client
WORKDIR /src/client/frp-client
RUN go build -o /out/frp-client .

FROM node:20-bookworm-slim AS webui-builder
WORKDIR /src/apps/client-webui
COPY apps/shared /src/apps/shared
COPY apps/client-webui/package*.json ./
ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY} \
    && npm config set fetch-retries 5 \
    && npm config set fetch-retry-maxtimeout 120000 \
    && npm config set fetch-timeout 120000
RUN --mount=type=cache,target=/root/.npm npm ci --prefer-offline --no-audit --fund=false --registry=${NPM_REGISTRY}
COPY apps/client-webui ./
RUN npm run build

FROM alpine:3.20
WORKDIR /opt/frp-client
COPY --from=go-builder /out/frp-client /opt/frp-client/frp-client
COPY --from=webui-builder /src/apps/client-webui/dist /opt/frp-client/webui
EXPOSE 18080
CMD ["/opt/frp-client/frp-client", "-addr", "0.0.0.0:18080", "-web", "/opt/frp-client/webui"]
