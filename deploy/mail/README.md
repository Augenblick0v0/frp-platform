# Mail Server

本目录用于 `docker-mailserver` 的持久化配置。

第一版目标是让平台可以用自己的域名发送验证码，例如：

```text
noreply@example.com
```

部署前需要在 DNS 添加：

```text
A     mail.example.com               <server-ip>
MX    example.com                    mail.example.com
TXT   example.com                    v=spf1 mx ~all
TXT   _dmarc.example.com             v=DMARC1; p=quarantine; rua=mailto:postmaster@example.com
TXT   default._domainkey.example.com DKIM 公钥，由 docker-mailserver 生成
```

创建邮箱账号示例：

```bash
docker exec -it frp-platform-mail setup email add noreply@example.com 'change-me'
docker exec -it frp-platform-mail setup config dkim
```

然后把 DKIM 公钥写入 DNS，并在 `.env` 中配置：

```env
SMTP_HOST=mail-server
SMTP_PORT=587
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=change-me
SMTP_FROM_EMAIL=noreply@example.com
```
