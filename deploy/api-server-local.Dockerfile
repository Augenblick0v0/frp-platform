FROM alpine:3.20
RUN apk add --no-cache certbot bind-tools
WORKDIR /app
COPY dist/fnos/api-server /app/api-server
ENV HTTP_ADDR=:8080
EXPOSE 8080
CMD ["/app/api-server"]
