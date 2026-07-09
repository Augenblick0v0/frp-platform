import React from 'react';
import { Card, Empty } from 'antd';

export function LogPanel({ title = '运行输出', value, height = 340 }) {
  return (
    <Card title={title} bordered={false} className="log-card">
      {value ? <pre className="log-panel" style={{ minHeight: height }}>{typeof value === 'string' ? value : JSON.stringify(value, null, 2)}</pre> : <Empty description="暂无输出" />}
    </Card>
  );
}
