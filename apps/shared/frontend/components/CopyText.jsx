import React from 'react';
import { Button, Typography, message } from 'antd';
import { CopyOutlined } from '@ant-design/icons';

export function CopyText({ text, children, type = 'link' }) {
  const copy = async () => {
    await navigator.clipboard.writeText(String(text ?? ''));
    message.success('已复制');
  };
  return <Button size="small" type={type} icon={<CopyOutlined />} onClick={copy}>{children || '复制'}</Button>;
}

export function CodeLine({ text }) {
  return <Typography.Text code copyable={{ text }}>{text || '-'}</Typography.Text>;
}
