import React from 'react';
import { Card, Steps, Typography } from 'antd';

export function RoleTopology({ mode = 'user', data }) {
  const items = mode === 'admin'
    ? [
      { title: 'Admin Console', description: '后台管理端' },
      { title: 'Master', description: 'API Server 控制面' },
      { title: 'Server(FRPS)', description: `${data?.online_nodes ?? 0}/${data?.node_count ?? 0} 节点在线` },
      { title: 'Client(FRPC) / Visitor', description: '本地客户端与外部访问者' },
    ]
    : [
      { title: 'User Console', description: '创建隧道与套餐支付' },
      { title: 'Master', description: '生成配置并校验套餐' },
      { title: 'Server(FRPS)', description: `${data?.nodes?.length ?? 0} 个安全节点` },
      { title: 'Client(FRPC) / Visitor', description: '本地连接与公网访问' },
    ];
  return (
    <Card className="topology-card" title="角色拓扑" extra={<Typography.Text type="secondary">frp-panel roles</Typography.Text>} bordered={false}>
      <Steps responsive current={mode === 'admin' ? 2 : 1} items={items} />
    </Card>
  );
}
