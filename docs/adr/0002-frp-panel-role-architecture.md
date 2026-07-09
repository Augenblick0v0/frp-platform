# ADR 0002: 对齐 frp-panel 角色架构

日期：2026-07-09

## 状态

Accepted

## 背景

平台原本以“后台、用户端、节点、客户端”等词汇描述系统，足够支持功能开发，但不利于说明三端边界、节点职责和用户本地 frpc 的位置。frp-panel wiki 使用 Master、Server(FRPS)、Client(FRPC)、Admin、Visitor 这些角色描述控制面、节点面、本地客户端和访问者之间的关系，适合作为本项目的统一架构语言。

## 决策

项目采用以下映射：

- `apps/api-server` 是 **Master Control Plane**。
- `apps/admin-web` 是 **Admin Console**。
- `apps/user-web` 是 **User Console**。
- `apps/node-agent` + `frps` + `node-nginx` 是 **Server(FRPS) Node Plane**。
- `client/frp-client` + `apps/client-webui` 是 **Client(FRPC)**。
- 访问公网入口的外部浏览器、App 或网络客户端是 **Visitor**。

控制面统一负责用户、套餐、订单、支付通道、兑换码、隧道、节点、证书、流量和 frpc 配置生成。节点面只负责承载 frps、node-agent、node-nginx 和公网入口。本地客户端只负责拉取配置、启动 frpc、提供本地日志和测速服务。

## 影响

- 后续文档、接口和前端页面使用这套角色语言。
- 用户控制台可以展示安全的拓扑摘要，但不能暴露 node-agent token、支付密钥、frps token 或后台专属字段。
- 后台管理端可以展示完整运维拓扑、节点操作结果和支付配置状态。
- Docker Compose 注释和部署文档按 Master / Admin Console / User Console / Server(FRPS) / Client(FRPC) 整理。

## 取舍

采用 frp-panel 角色语言会增加少量术语学习成本，但能显著降低“后台容器、用户控制台容器、本地客户端、节点容器”之间的边界混淆，特别适合后续多节点和本地客户端分发。
