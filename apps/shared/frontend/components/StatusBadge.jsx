import React from 'react';
import { Badge, Tag } from 'antd';

export function StatusBadge({ status, text }) {
  const s = String(status || 'unknown').toLowerCase();
  let color = 'blue';
  let badgeStatus = 'processing';
  if (/(active|online|paid|success|running|issued|ok)/.test(s)) { color = 'green'; badgeStatus = 'success'; }
  else if (/(pending|created|checking|dry_run|renewing)/.test(s)) { color = 'orange'; badgeStatus = 'warning'; }
  else if (/(error|fail|deleted|expired|inactive|disabled|stopped|replaced)/.test(s)) { color = 'red'; badgeStatus = 'error'; }
  return <Tag color={color}><Badge status={badgeStatus} text={text || status || 'unknown'} /></Tag>;
}
