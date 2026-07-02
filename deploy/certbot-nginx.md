# Nginx / Let's Encrypt 自动化说明

API Server 提供这些管理接口：

```text
POST /api/admin/domains/check-cname
POST /api/admin/nginx/render-https
POST /api/admin/nginx/test
POST /api/admin/nginx/reload
POST /api/admin/certificates/request
```

默认部署中：

- API Server 挂载 `./nginx/conf.d` 到 `/app/runtime/nginx-conf.d`
- API Server 挂载 `./certbot/www` 到 `/var/www/certbot`
- API Server 挂载 `./certbot/letsencrypt` 到 `/etc/letsencrypt`
- Nginx 同样挂载这些目录

流程：

```text
1. 用户把 app.user.com CNAME 到 frp.example.com
2. 后台调用 CNAME 检测
3. 后台调用证书申请接口
4. 后台生成 HTTPS Nginx 配置
5. 后台执行 Nginx test/reload
6. 用户访问 https://app.user.com，不带端口
```

`.env` 中默认 `CERTBOT_DRY_RUN=true`。正式申请证书前改为：

```env
CERTBOT_DRY_RUN=false
LETSENCRYPT_EMAIL=admin@example.com
```

`NGINX_TEST_CMD` 和 `NGINX_RELOAD_CMD` 默认留空，表示只生成配置，不执行 reload。生产环境可以按部署方式配置，例如把 Docker socket 安全挂载给一个受限 reload sidecar 后执行 reload。
