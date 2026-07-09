import React from 'react';
import { Card, Descriptions, Typography } from 'antd';
import { LogPanel } from './LogPanel.jsx';
import { formatTime } from '../api/client.js';

export function NodeOperationPanel({ operation }) {
  return (
    <Card title="Node Operation Panel" bordered={false} className="node-operation-panel">
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="当前节点">{operation?.nodeName || '-'}</Descriptions.Item>
        <Descriptions.Item label="当前动作">{operation?.action || '-'}</Descriptions.Item>
        <Descriptions.Item label="执行时间">{formatTime(operation?.time)}</Descriptions.Item>
        <Descriptions.Item label="状态">{operation?.status || '等待操作'}</Descriptions.Item>
      </Descriptions>
      <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>节点状态、配置、日志、重启、reload、nginx test/reload 的结果都会在这里显示。</Typography.Paragraph>
      <LogPanel title="返回内容" value={operation?.output} height={260} />
    </Card>
  );
}
