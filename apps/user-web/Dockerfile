FROM node:20-bookworm-slim AS frontend-builder
WORKDIR /src/apps/user-web
COPY apps/shared /src/apps/shared
COPY apps/user-web/package*.json ./
ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY} \
    && npm config set fetch-retries 5 \
    && npm config set fetch-retry-maxtimeout 120000 \
    && npm config set fetch-timeout 120000
RUN --mount=type=cache,target=/root/.npm npm ci --prefer-offline --no-audit --fund=false --registry=${NPM_REGISTRY}
COPY apps/user-web ./
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=frontend-builder /src/apps/user-web/dist /usr/share/nginx/html
COPY dist /usr/share/nginx/html/downloads
COPY apps/user-web/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
