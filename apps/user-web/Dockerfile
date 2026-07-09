FROM nginx:1.27-alpine
COPY apps/user-web /usr/share/nginx/html
COPY dist /usr/share/nginx/html/downloads
COPY apps/user-web/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
