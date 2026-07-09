
import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert, Button, Card, Col, Descriptions, Divider, Drawer, Empty, Form, Input, InputNumber, Modal, Progress,
  Radio, Row, Select, Space, Steps, Table, Tabs, Tag, Typography, message
} from 'antd';
import {
  AppstoreOutlined, ThunderboltOutlined, DeploymentUnitOutlined, PlusCircleOutlined, CloudServerOutlined,
  DesktopOutlined, SafetyCertificateOutlined, WalletOutlined, GiftOutlined, UserOutlined, QuestionCircleOutlined,
  RocketOutlined, CopyOutlined
} from '@ant-design/icons';
import { AppShell } from '../../shared/frontend/components/AppShell.jsx';
import { MetricCard } from '../../shared/frontend/components/MetricCard.jsx';
import { RoleTopology } from '../../shared/frontend/components/RoleTopology.jsx';
import { StatusBadge } from '../../shared/frontend/components/StatusBadge.jsx';
import { LogPanel } from '../../shared/frontend/components/LogPanel.jsx';
import { CopyText, CodeLine } from '../../shared/frontend/components/CopyText.jsx';
import { ApiClient, formatBytes, formatMbps, formatMoney, formatTime, percent } from '../../shared/frontend/api/client.js';

const api = new ApiClient({ tokenKey: 'token' });
const routes = {
  overview: { title: '总览', subtitle: '账户、套餐、流量和资源概览' },
  quickstart: { title: '快速开始', subtitle: '按步骤完成客户端下载、配置同步和隧道访问' },
  tunnels: { title: '隧道列表', subtitle: '管理 HTTP / HTTPS / TCP / UDP 隧道' },
  create: { title: '创建隧道', subtitle: '按协议、节点、本地服务和公网入口逐步创建' },
  speedtest: { title: '隧道测速', subtitle: 'API Server 发起真实测速流程' },
  nodes: { title: '节点状态', subtitle: '查看可用 Server(FRPS) 节点安全字段' },
  client: { title: '客户端 FRPC', subtitle: '下载本地 Windows / Linux 客户端并同步配置' },
  domains: { title: '域名证书', subtitle: '提交自定义域名和 HTTPS 证书申请' },
  billing: { title: '套餐支付', subtitle: '选择套餐并使用微信支付或支付宝下单' },
  redeem: { title: '兑换码', subtitle: '兑换已绑定套餐的激活码' },
  account: { title: '用户中心', subtitle: '账户、Token 和当前套餐信息' },
  help: { title: '帮助文档', subtitle: '了解 Master / Server(FRPS) / Client(FRPC) / Visitor 工作方式' },
};

function hashRoute() { return location.hash.replace(/^#\/?/, '') || 'overview'; }
function apiOrigin() { return window.location.origin.replace(/\/$/, ''); }
function protocolEnabled(sub, type) {
  if (!sub || sub.status !== 'active') return false;
  return Boolean(sub[`allow_${type}`]);
}
function planName(plans, id) { return plans.find(p => Number(p.id) === Number(id))?.name || `#${id}`; }

function AuthPage({ mode, onAuthed }) {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();
  const isRegister = mode === 'register';
  async function submit(values) {
    setLoading(true);
    try {
      if (isRegister) {
        await api.post('/api/auth/register', values, { token: '' });
        message.success('注册成功，请登录');
        location.hash = 'login';
      } else {
        const data = await api.post('/api/auth/login', values, { token: '' });
        api.setToken(data.access_token);
        onAuthed?.();
      }
    } catch (err) { message.error(err.message); }
    finally { setLoading(false); }
  }
  async function sendCode() {
    try {
      const email = form.getFieldValue('email');
      const data = await api.post('/api/auth/send-email-code', { email, purpose: 'register' }, { token: '' });
      if (data?.dev_code) form.setFieldValue('code', data.dev_code);
      message.success('验证码已发送');
    } catch (err) { message.error(err.message); }
  }
  return <div className="auth-layout">
    <section className="auth-hero">
      <div className="hero-dots"><i /></div>
      <Typography.Text type="secondary">FRP Tunnel Platform</Typography.Text>
      <Typography.Title>{isRegister ? '创建账户，开始使用隧道' : '登录用户控制台'}</Typography.Title>
      <Typography.Paragraph>参考 mefrp 的工具面板动线，将套餐、隧道、节点、客户端和测速放在同一工作台中。</Typography.Paragraph>
      <div className="quick-flow">
        <div>01 Master 校验套餐权限</div><div>02 Client(FRPC) 同步配置</div><div>03 Visitor 访问公网入口</div>
      </div>
    </section>
    <section className="auth-card-wrap">
      <Card title={isRegister ? '注册' : '登录'} extra={<Button type="link" onClick={() => { location.hash = isRegister ? 'login' : 'register'; }}>{isRegister ? '去登录' : '去注册'}</Button>}>
        <Form form={form} layout="vertical" onFinish={submit}>
          <Form.Item name="email" label="邮箱" rules={[{ required: true }]}><Input autoComplete="email" /></Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true }]}><Input.Password /></Form.Item>
          {isRegister && <Form.Item label="邮箱验证码" required><Space.Compact style={{ width: '100%' }}><Form.Item name="code" noStyle rules={[{ required: true }]}><Input /></Form.Item><Button onClick={sendCode}>发送</Button></Space.Compact></Form.Item>}
          <Button type="primary" htmlType="submit" loading={loading} block>{isRegister ? '创建账户' : '登录控制台'}</Button>
        </Form>
      </Card>
    </section>
  </div>;
}

export default function App() {
  const [route, setRoute] = useState(hashRoute());
  const [state, setState] = useState({ loading: true, me: null, topology: null, sub: null, traffic: null, tunnels: [], plans: [], nodes: [], clientConfig: null });
  const [log, setLog] = useState('ready');
  useEffect(() => { const f = () => setRoute(hashRoute()); window.addEventListener('hashchange', f); return () => window.removeEventListener('hashchange', f); }, []);
  const authed = Boolean(api.token());
  async function load() {
    if (!api.token()) { setState(s => ({ ...s, loading: false, me: null })); return; }
    setState(s => ({ ...s, loading: true }));
    try {
      const [me, topology, sub, traffic, tunnels, plans, nodes, clientConfig] = await Promise.all([
        api.get('/api/auth/me'), api.get('/api/user/topology').catch(() => null), api.get('/api/user/subscription').catch(() => null), api.get('/api/user/traffic').catch(() => null), api.get('/api/tunnels').catch(() => []), api.get('/api/user/plans').catch(() => []), api.get('/api/user/nodes').catch(() => []), api.get('/api/client/tunnels').catch(() => null)
      ]);
      setState({ loading: false, me, topology, sub: sub || topology?.subscription, traffic: traffic || topology?.traffic, tunnels, plans, nodes: nodes?.length ? nodes : (topology?.nodes || []), clientConfig });
    } catch (err) { api.setToken(''); setState(s => ({ ...s, loading: false, me: null })); location.hash = 'login'; }
  }
  useEffect(() => { load(); }, []);
  if (!authed && (route === 'login' || route === 'register')) return <AuthPage mode={route} onAuthed={() => { location.hash = 'overview'; load(); }} />;
  if (!authed) return <Landing />;
  const current = routes[route] ? route : 'overview';
  const menuItems = [
    { type: 'group', label: '概览', children: [item('overview', '总览', <AppstoreOutlined />), item('quickstart', '快速开始', <RocketOutlined />, '快速开始')] },
    { type: 'group', label: '隧道', children: [item('tunnels', '隧道列表', <DeploymentUnitOutlined />, '隧道列表'), item('create', '创建隧道', <PlusCircleOutlined />, '创建隧道'), item('speedtest', '隧道测速', <ThunderboltOutlined />, '隧道测速')] },
    { type: 'group', label: '资源', children: [item('nodes', '节点状态', <CloudServerOutlined />, '节点状态'), item('client', '客户端 FRPC', <DesktopOutlined />, '客户端'), item('domains', '域名证书', <SafetyCertificateOutlined />, '域名证书')] },
    { type: 'group', label: '账户', children: [item('billing', '套餐支付', <WalletOutlined />, '套餐支付'), item('redeem', '兑换码', <GiftOutlined />), item('account', '用户中心', <UserOutlined />, '用户中心'), item('help', '帮助文档', <QuestionCircleOutlined />, '帮助文档')] },
  ];
  return <AppShell title={routes[current].title} subtitle={routes[current].subtitle} brand="FRP User" brandSub="Master + Client" menuItems={menuItems} selectedKey={current} onSelect={k => { location.hash = k; setRoute(k); }} onRefresh={load} userLabel={state.me?.email} onLogout={() => { api.setToken(''); location.hash = 'login'; }} storageKey="userSidebarCollapsedV4">
    {renderPage(current, state, load, log, setLog)}
  </AppShell>;
}
function item(key, label, icon, shortLabel) { return { key, icon, label, rawLabel: label, shortLabel: shortLabel || label }; }

function Landing() { return <div className="public-landing"><div className="landing-card"><div className="landing-copy"><Typography.Text type="secondary">FRP Tunnel Platform</Typography.Text><Typography.Title>蓝白工具面板风的内网穿透控制台</Typography.Title><Typography.Paragraph>后台、用户控制台和本地客户端分离，对齐 Master / Server(FRPS) / Client(FRPC) / Visitor 架构。</Typography.Paragraph><Space><Button type="primary" size="large" onClick={() => location.hash = 'login'}>登录</Button><Button size="large" onClick={() => location.hash = 'register'}>注册</Button></Space></div><div className="landing-side"><Typography.Title style={{ color: '#fff' }} level={3}>快速动线</Typography.Title><div className="quick-flow"><div>开通套餐</div><div>下载 Client(FRPC)</div><div>创建隧道</div><div>Visitor 访问公网入口</div></div></div></div></div>; }

function renderPage(route, state, load, log, setLog) {
  const props = { state, load, log, setLog };
  return ({ overview: <Overview {...props} />, quickstart: <QuickStart {...props} />, tunnels: <Tunnels {...props} />, create: <CreateTunnel {...props} />, speedtest: <SpeedTest {...props} />, nodes: <Nodes {...props} />, client: <Client {...props} />, domains: <Domains {...props} />, billing: <Billing {...props} />, redeem: <Redeem {...props} />, account: <Account {...props} />, help: <Help /> })[route] || <Overview {...props} />;
}

function Overview({ state }) {
  const sub = state.sub || {};
  const traffic = state.traffic || {};
  return <div className="panel-stack">
    <Alert showIcon type={sub.status === 'active' ? 'success' : 'warning'} message={sub.status === 'active' ? `当前套餐：${sub.plan_name}` : '尚未开通有效套餐'} description="Client(FRPC) 从 Master 拉取配置，再连接 Server(FRPS) 节点。" />
    <div className="metric-grid"><MetricCard title="套餐状态" value={sub.status || 'inactive'} trend={sub.plan_name || '-'} /><MetricCard title="剩余流量" value={formatBytes(traffic.traffic_left_bytes)} trend={`${formatBytes(traffic.traffic_used_bytes)} / ${formatBytes(traffic.traffic_limit_bytes)}`} /><MetricCard title="隧道数" value={state.tunnels.length} trend={`TCP ${state.topology?.tunnel_counts?.tcp || 0} / HTTP ${state.topology?.tunnel_counts?.http || 0}`} /><MetricCard title="在线节点" value={(state.nodes || []).filter(n => n.status === 'online').length} trend={`共 ${state.nodes.length} 个`} /></div>
    <RoleTopology mode="user" data={state.topology} />
    <div className="two-column"><Card title="最近隧道" bordered={false}><TunnelTable data={state.tunnels.slice(0, 6)} /></Card><Card title="快捷入口" bordered={false}><Space direction="vertical" style={{ width: '100%' }}><Button block onClick={() => location.hash = 'create'}>创建隧道</Button><Button block onClick={() => location.hash = 'client'}>下载客户端</Button><Button block onClick={() => location.hash = 'billing'}>购买套餐</Button><Button block onClick={() => location.hash = 'speedtest'}>执行测速</Button></Space></Card></div>
  </div>;
}
function QuickStart({ state }) { return <Card bordered={false} className="page-card"><Steps direction="vertical" current={state.sub?.status === 'active' ? 2 : 0} items={[{ title: '开通套餐', description: '购买套餐或兑换码激活' }, { title: '下载客户端', description: 'Windows / Linux Client(FRPC)' }, { title: '创建隧道', description: '选择协议、节点、本地服务和公网入口' }, { title: '同步配置', description: '客户端从 Master 拉取 frpc 配置' }, { title: '访问入口', description: 'Visitor -> Server(FRPS) -> Client(FRPC) -> Local Service' }]} /></Card>; }
function TunnelTable({ data }) { return <Table size="middle" rowKey="id" dataSource={data} pagination={data.length > 8 ? { pageSize: 8 } : false} columns={[{ title: 'ID', dataIndex: 'id', width: 70, render: v => <Typography.Text type="secondary">#{v}</Typography.Text> }, { title: '名称', dataIndex: 'name' }, { title: '协议', dataIndex: 'type', render: v => <Tag color="blue">{v}</Tag> }, { title: '公网入口', dataIndex: 'public_url', render: v => v ? <CodeLine text={v} /> : '-' }, { title: '本地服务', render: (_, r) => `${r.local_host}:${r.local_port}` }, { title: '状态', dataIndex: 'status', render: v => <StatusBadge status={v} /> }]} />; }
function Tunnels({ state }) { const [open, setOpen] = useState(null); return <Card bordered={false} className="page-card"><div className="table-toolbar"><Alert type="info" showIcon message="表格 + Drawer 动线" description="点击行可查看隧道详情和客户端配置要点。" /><Button type="primary" onClick={() => location.hash = 'create'}>创建隧道</Button></div><Table rowKey="id" dataSource={state.tunnels} onRow={r => ({ onClick: () => setOpen(r) })} columns={[{ title: 'ID', dataIndex: 'id', width: 70 }, { title: '名称', dataIndex: 'name' }, { title: '协议', dataIndex: 'type', render: v => <Tag color="blue">{v}</Tag> }, { title: '公网入口', dataIndex: 'public_url', render: v => v ? <CodeLine text={v} /> : '-' }, { title: '限速', dataIndex: 'effective_bandwidth_limit_kbps', render: formatMbps }, { title: '创建时间', dataIndex: 'created_at', render: formatTime }, { title: '状态', dataIndex: 'status', render: v => <StatusBadge status={v} /> }]} /><Drawer open={Boolean(open)} onClose={() => setOpen(null)} title="隧道详情" width={520}>{open && <Descriptions column={1} bordered items={Object.entries(open).map(([k, v]) => ({ key: k, label: k, children: typeof v === 'object' ? JSON.stringify(v) : String(v ?? '-') }))} />}</Drawer></Card>; }

function CreateTunnel({ state, load }) {
  const [form] = Form.useForm(); const [step, setStep] = useState(0); const [protocol, setProtocol] = useState('tcp'); const sub = state.sub || {};
  async function submit() { try { const values = await form.validateFields(); const payload = { ...values, type: protocol, bandwidth_limit_kbps: values.bandwidth_limit_kbps || 0 }; const t = await api.post('/api/tunnels', payload); message.success('隧道已创建'); await load(); Modal.success({ title: '创建成功', content: t.public_url }); location.hash = 'tunnels'; } catch (err) { if (err?.errorFields) return; message.error(err.message); } }
  const protocols = ['tcp','udp','http','https'];
  return <div className="two-column"><Card bordered={false} className="page-card"><Steps current={step} direction="vertical" items={[{ title: '选择协议' }, { title: '选择 FRPS 节点' }, { title: '填写本地服务' }, { title: '配置公网入口' }, { title: '确认创建' }]} /></Card><Card bordered={false} className="page-card"><Alert type="info" showIcon message="Client(FRPC) 将从 Master 拉取配置，并连接所选 Server(FRPS)。" style={{ marginBottom: 16 }} /><Form layout="vertical" form={form} initialValues={{ local_host: '127.0.0.1', local_port: 80 }}>
    {step === 0 && <div className="protocol-grid">{protocols.map(p => <div key={p} className={`protocol-card ${protocol===p?'active':''}`} onClick={() => { setProtocol(p); setStep(1); }}><Typography.Title level={4}>{p.toUpperCase()}</Typography.Title><Typography.Text type={protocolEnabled(sub, p) ? 'success' : 'secondary'}>{protocolEnabled(sub, p) ? '已开通' : '需要套餐权限'}</Typography.Text></div>)}</div>}
    {step === 1 && <Form.Item name="node_id" label="FRPS 节点"><Select options={(state.nodes || []).map(n => ({ value: n.id, label: `${n.name} / ${n.server_addr || n.frp_entry_domain || '-'}` }))} onChange={() => setStep(2)} /></Form.Item>}
    {step === 2 && <><Form.Item name="name" label="隧道名称" rules={[{ required: true }]}><Input /></Form.Item><Row gutter={12}><Col span={12}><Form.Item name="local_host" label="Local Host" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="local_port" label="Local Port" rules={[{ required: true }]}><InputNumber min={1} max={65535} style={{ width: '100%' }} /></Form.Item></Col></Row><Button type="primary" onClick={() => setStep(3)}>下一步</Button></>}
    {step === 3 && <><Form.Item name="domain" label="自定义域名" tooltip="HTTP/HTTPS 需要域名，TCP/UDP 由 Master 分配端口"><Input disabled={!['http','https'].includes(protocol)} placeholder="app.example.com" /></Form.Item><Form.Item name="bandwidth_limit_kbps" label="隧道降速覆盖（Kbps）"><InputNumber min={0} max={sub.bandwidth_limit_kbps || undefined} style={{ width: '100%' }} /></Form.Item><Button type="primary" onClick={() => setStep(4)}>下一步</Button></>}
    {step === 4 && <><Descriptions bordered column={1} items={[{ label: '协议', children: protocol }, { label: '套餐限速', children: formatMbps(sub.bandwidth_limit_kbps) }, { label: '说明', children: 'Visitor -> Server(FRPS) -> Client(FRPC) -> Local Service' }]} /><Divider /><Space><Button onClick={() => setStep(0)}>重选</Button><Button type="primary" onClick={submit}>创建</Button></Space></>}
  </Form></Card></div>;
}

function Nodes({ state }) { return <div className="card-grid">{state.nodes.map(n => <Card key={n.id} bordered={false} className="page-card" title={n.name} extra={<StatusBadge status={n.status} />}><Descriptions column={1} size="small" items={[{ label: '入口域名', children: n.frp_entry_domain || '-' }, { label: 'frps', children: `${n.server_addr || '-'}:${n.frp_server_port || '-'}` }, { label: 'TCP', children: `${n.tcp_port_start}-${n.tcp_port_end}` }, { label: 'UDP', children: `${n.udp_port_start}-${n.udp_port_end}` }, { label: '最后在线', children: formatTime(n.last_seen_at) }]} /></Card>)}</div>; }
function Client({ state }) { const token = api.token(); return <div className="two-column"><Card title="客户端下载" bordered={false}>{(state.topology?.downloads || []).map(d => <Card key={d.platform} size="small" style={{ marginBottom: 12 }}><Space direction="vertical"><Typography.Title level={5}>{d.label}</Typography.Title><a href={d.url}>{d.url}</a></Space></Card>)}</Card><Card title="连接信息" bordered={false}><Descriptions column={1} bordered items={[{ label: 'API Server', children: <CodeLine text={apiOrigin()} /> }, { label: 'Token', children: <Space><Typography.Text code ellipsis style={{ maxWidth: 210 }}>{token}</Typography.Text><CopyText text={token}>Token</CopyText></Space> }, { label: 'frps', children: `${state.clientConfig?.server_addr || '-'}:${state.clientConfig?.server_port || '-'}` }]} /></Card></div>; }
function Domains() { const [form] = Form.useForm(); async function submit(v){ try { const res = await api.post('/api/user/certificates/request', v); Modal.info({ title: '申请结果', content: <pre>{JSON.stringify(res, null, 2)}</pre> }); } catch(e){ message.error(e.message); } } return <Card bordered={false} className="page-card"><Alert showIcon type="info" message="域名证书按需保留" description="用户提交申请，后台可统一检查 CNAME、生成 Nginx 配置和续期。" style={{ marginBottom: 16 }} /><Form form={form} layout="vertical" onFinish={submit}><Form.Item name="domain" label="域名" rules={[{ required: true }]}><Input placeholder="app.example.com" /></Form.Item><Form.Item name="email" label="证书邮箱"><Input /></Form.Item><Button type="primary" htmlType="submit">申请证书</Button></Form></Card>; }
function Billing({ state, load }) { const [payType, setPayType] = useState('wxpay'); const [pay, setPay] = useState(null); async function buy(plan){ try { const order = await api.post('/api/payments/epay/orders', { plan_id: plan.id, pay_type: payType }); setPay(order); message.success('订单已创建'); await load(); } catch(e){ message.error(e.message); } } return <div className="panel-stack"><Alert showIcon type="info" message="当前支付方式" description="微信支付 -> pay_type=wxpay -> 通道 wxpay_zg。" /><Radio.Group value={payType} onChange={e => setPayType(e.target.value)}><Radio.Button value="wxpay">微信支付</Radio.Button><Radio.Button value="alipay">支付宝</Radio.Button></Radio.Group><div className="plan-grid">{state.plans.map(p => <Card className="plan-card" key={p.id} title={p.name} bordered={false} extra={<Tag>{formatMoney(p.price_cents)}</Tag>}><Space direction="vertical" style={{ width: '100%' }}><Typography.Paragraph>{p.description}</Typography.Paragraph><Descriptions size="small" column={1} items={[{ label: '时长', children: `${p.duration_days} 天` }, { label: '流量', children: formatBytes(p.traffic_limit_bytes) }, { label: '带宽', children: formatMbps(p.bandwidth_limit_kbps) }, { label: '隧道', children: p.max_tunnels }]} /><Button type="primary" block onClick={() => buy(p)}>购买</Button></Space></Card>)}</div>{pay && <Card title="支付订单" bordered={false}><Descriptions column={1} bordered items={[{ label: '订单号', children: pay.out_trade_no }, { label: '金额', children: pay.money }, { label: '方式', children: pay.pay_type }, { label: '支付链接', children: <a target="_blank" href={pay.pay_url}>{pay.pay_url}</a> }]} /></Card>}</div>; }
function Redeem({ load }) { const [form] = Form.useForm(); const [result, setResult] = useState(null); async function submit(v){ try { const res = await api.post('/api/user/redeem', v); setResult(res); message.success('兑换成功'); await load(); } catch(e){ message.error(e.message); } } return <Card bordered={false} className="page-card"><Alert showIcon type="info" message="兑换码已绑定套餐" style={{ marginBottom: 16 }} /><Form form={form} layout="inline" onFinish={submit}><Form.Item name="code" rules={[{ required: true }]}><Input placeholder="DEMO-PLAN-2026" /></Form.Item><Button type="primary" htmlType="submit">兑换</Button></Form>{result && <Card style={{ marginTop: 16 }}><Descriptions column={1} bordered items={[{ label: '套餐', children: result.plan_name }, { label: '到期', children: formatTime(result.expires_at) }, { label: '状态', children: <StatusBadge status={result.status} /> }]} /></Card>}</Card>; }
function SpeedTest({ state, log, setLog, load }) { const [form] = Form.useForm(); const [current, setCurrent] = useState(0); const [running, setRunning] = useState(false); async function local(base, path, body, localToken){ const headers = { 'Content-Type': 'application/json' }; if (localToken) headers['X-Local-Token'] = localToken; const res = await fetch(base.replace(/\/$/,'') + path, { method: 'POST', headers, body: JSON.stringify(body || {}) }); const json = await res.json(); if(!res.ok || json.success === false) throw new Error(json.message || res.statusText); return json.data; } async function run(v){ setRunning(true); setLog('start'); try { setCurrent(0); localStorage.setItem('localClientToken', v.local_token || ''); const bench = await local(v.local_client, '/api/speed-tests/prepare', { type: v.type }, v.local_token); setLog(l => l + '\nprepared ' + JSON.stringify(bench)); setCurrent(1); const tunnel = await api.post('/api/speed-tests/tunnels', { type: v.type, local_host: bench.host, local_port: bench.port, node_id: v.node_id || 0 }); setLog(l => l + '\ntunnel ' + JSON.stringify(tunnel)); setCurrent(2); await local(v.local_client, '/api/config/sync', { api_base: apiOrigin(), token: api.token(), speed_test_id: tunnel.id }, v.local_token); setCurrent(3); await local(v.local_client, '/api/frpc/restart', {}, v.local_token); setCurrent(4); const result = await api.post(`/api/speed-tests/${tunnel.id}/run`, { download_bytes: (v.download_mb || 4) * 1024 * 1024, upload_bytes: (v.upload_mb || 2) * 1024 * 1024, duration_seconds: v.duration_seconds || 45 }); setCurrent(6); setLog(l => l + '\nresult ' + JSON.stringify(result, null, 2)); await local(v.local_client, '/api/speed-tests/cleanup', {}, v.local_token).catch(()=>{}); await load(); } catch(e){ message.error(e.message); setLog(l => l + '\nERROR ' + e.message); } finally { setRunning(false); } } return <div className="two-column"><Card bordered={false} className="page-card" title="测速参数"><Form layout="vertical" form={form} onFinish={run} initialValues={{ local_client: localStorage.getItem('localClientBase') || 'http://127.0.0.1:18080', local_token: localStorage.getItem('localClientToken') || '', type: 'tcp', download_mb: 4, upload_mb: 2, duration_seconds: 45 }}><Form.Item name="local_client" label="本地客户端 API"><Input onChange={e => localStorage.setItem('localClientBase', e.target.value)} /></Form.Item><Form.Item name="local_token" label="Local Client Token" rules={[{ required: true, message: '请输入本地客户端 X-Local-Token' }]}><Input.Password placeholder="从本地客户端 WebUI 获取 X-Local-Token" onChange={e => localStorage.setItem('localClientToken', e.target.value)} /></Form.Item><Form.Item name="node_id" label="节点"><Select allowClear options={state.nodes.map(n => ({ value: n.id, label: n.name }))} /></Form.Item><Form.Item name="type" label="协议"><Select options={['tcp','udp','http','https'].map(v => ({ value: v, label: v.toUpperCase() }))} /></Form.Item><Row gutter={12}><Col span={12}><Form.Item name="download_mb" label="Download MB"><InputNumber min={1} style={{ width:'100%' }} /></Form.Item></Col><Col span={12}><Form.Item name="upload_mb" label="Upload MB"><InputNumber min={1} style={{ width:'100%' }} /></Form.Item></Col></Row><Button type="primary" htmlType="submit" loading={running} block>开始测速</Button></Form></Card><div className="panel-stack"><Card bordered={false} className="page-card"><Steps className="speed-steps" current={current} direction="vertical" items={['本地准备 benchmark','Master 创建临时隧道','Client 同步临时配置','frpc 重启连接','API Server 发起测速','恢复正式配置','清理临时服务'].map(title => ({ title }))} /></Card><LogPanel value={log} /></div></div>; }
function Account({ state }) { const sub = state.sub || {}; const traffic = state.traffic || {}; return <div className="two-column"><Card title="账户" bordered={false}><Descriptions column={1} bordered items={[{ label: 'Email', children: state.me?.email }, { label: 'Token', children: <Space><Typography.Text code ellipsis style={{ maxWidth: 220 }}>{api.token()}</Typography.Text><CopyText text={api.token()} /></Space> }, { label: '状态', children: <StatusBadge status={state.me?.status} /> }]} /></Card><Card title="套餐" bordered={false}><Progress percent={percent(traffic.traffic_used_bytes, traffic.traffic_limit_bytes)} /><Descriptions column={1} items={[{ label: '套餐', children: sub.plan_name || '-' }, { label: '到期', children: formatTime(sub.expires_at) }, { label: '流量', children: `${formatBytes(traffic.traffic_used_bytes)} / ${formatBytes(traffic.traffic_limit_bytes)}` }]} /></Card></div>; }
function Help() { return <div className="help-grid"><Card title="Master"><p>API Server 是控制面，负责用户、套餐、订单、隧道和 frpc 配置生成。</p></Card><Card title="Server(FRPS)"><p>节点承载 frps、node-agent 和 node-nginx，提供公网入口。</p></Card><Card title="Client(FRPC)"><p>本地 Windows/Linux 客户端拉取配置，启动 frpc，并提供测速 benchmark。</p></Card><Card title="Visitor"><p>Visitor 通过 Server(FRPS) 的公网入口访问用户本地服务。</p></Card></div>; }
