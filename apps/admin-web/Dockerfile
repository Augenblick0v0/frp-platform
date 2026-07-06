FROM nginx:1.27-alpine
COPY apps/admin-web /usr/share/nginx/html
COPY apps/admin-web/nginx.conf.template /etc/nginx/templates/default.conf.template
EXPOSE 80
