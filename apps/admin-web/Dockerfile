FROM node:20-bookworm-slim AS frontend-builder
WORKDIR /src/apps/admin-web
COPY apps/shared /src/apps/shared
COPY apps/admin-web/package*.json ./
ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY} \
    && npm config set fetch-retries 5 \
    && npm config set fetch-retry-maxtimeout 120000 \
    && npm config set fetch-timeout 120000
RUN --mount=type=cache,target=/root/.npm npm ci --prefer-offline --no-audit --fund=false --registry=${NPM_REGISTRY}
COPY apps/admin-web ./
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=frontend-builder /src/apps/admin-web/dist /usr/share/nginx/html
COPY apps/admin-web/nginx.conf.template /etc/nginx/templates/default.conf.template
EXPOSE 80
