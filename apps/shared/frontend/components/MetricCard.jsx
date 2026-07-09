import React from 'react';
import { Card, Statistic, Typography } from 'antd';

export function MetricCard({ title, value, suffix, prefix, icon, trend, children }) {
  return (
    <Card className="metric-card" bordered={false}>
      <div className="metric-head"><span>{icon}</span><Typography.Text type="secondary">{title}</Typography.Text></div>
      <Statistic value={value} suffix={suffix} prefix={prefix} valueStyle={{ fontSize: 25, color: '#0f172a' }} />
      {trend && <div className="metric-trend">{trend}</div>}
      {children}
    </Card>
  );
}
