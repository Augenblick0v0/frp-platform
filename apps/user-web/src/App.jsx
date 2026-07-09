
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
import { StatusBadge } from '../../shared/frontend/components/StatusBadge.jsx';
import { LogPanel } from '../../shared/frontend/components/LogPanel.jsx';
import { CopyText, CodeLine } from '../../shared/frontend/components/CopyText.jsx';
import { ApiClient, formatBytes, formatMbps, formatMoney, formatTime, percent } from '../../shared/frontend/api/client.js';

const api = new ApiClient({ tokenKey: 'token' });
const routes = {
  overview: { title: '控制台首页', subtitle: '资源状态、快捷操作和最近隧道' },
  quickstart: { title: '快速开始', subtitle: '开通套餐、安装客户端、创建隧道' },
  tunnels: { title: '隧道列表', subtitle: '查看和管理你的公网访问入口' },
  create: { title: '创建隧道', subtitle: '像填写表单一样创建公网访问入口' },
  speedtest: { title: '隧道测速', subtitle: '检测本地客户端到节点的实际连通质量' },
  nodes: { title: '节点状态', subtitle: '查看可用节点和入口端口范围' },
  client: { title: '客户端下载', subtitle: '下载本地客户端并同步隧道配置' },
  domains: { title: '域名证书', subtitle: '申请自定义域名和 HTTPS 证书' },
  billing: { title: '套餐支付', subtitle: '选择套餐并使用微信支付或支付宝下单' },
  redeem: { title: '兑换码', subtitle: '兑换已绑定套餐的激活码' },
  account: { title: '用户中心', subtitle: '账户、Token 和当前套餐信息' },
  help: { title: '帮助文档', subtitle: '常见使用问题和操作指引' },
};

function hashRoute() { return location.hash.replace(/^#\/?/, '') || 'overview'; }
function apiOrigin() { return window.location.origin.replace(/\/$/, ''); }
function protocolEnabled(sub, type) {
  if (!sub || sub.status !== 'active') return false;
  return Boolean(sub[`allow_${type}`]);
}
function planName(plans, id) { return plans.find(p => Number(p.id) === Number(id))?.name || `#${id}`; }

function subscriptionBandwidth(sub) {
  return Number(sub?.bandwidth_limit_kbps ?? sub?.bandwidth_kbps ?? 0);
}

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
      <Typography.Paragraph>一个面向日常使用的内网穿透控制台：查看套餐、创建隧道、下载客户端和执行测速。</Typography.Paragraph>
      <div className="quick-flow">
        <div>01 开通套餐</div><div>02 创建隧道</div><div>03 启动本地客户端</div>
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
    { type: 'group', label: '资源', children: [item('nodes', '节点状态', <CloudServerOutlined />, '节点状态'), item('client', '客户端下载', <DesktopOutlined />, '客户端'), item('domains', '域名证书', <SafetyCertificateOutlined />, '域名证书')] },
    { type: 'group', label: '账户', children: [item('billing', '套餐支付', <WalletOutlined />, '套餐支付'), item('redeem', '兑换码', <GiftOutlined />), item('account', '用户中心', <UserOutlined />, '用户中心'), item('help', '帮助文档', <QuestionCircleOutlined />, '帮助文档')] },
  ];
  return <AppShell title={routes[current].title} subtitle={routes[current].subtitle} brand="FRP User" brandSub="用户控制台" menuItems={menuItems} selectedKey={current} onSelect={k => { location.hash = k; setRoute(k); }} onRefresh={load} userLabel={state.me?.email} onLogout={() => { api.setToken(''); location.hash = 'login'; }} storageKey="userSidebarCollapsedV4">
    {renderPage(current, state, load, log, setLog)}
  </AppShell>;
}
function item(key, label, icon, shortLabel) { return { key, icon, label, rawLabel: label, shortLabel: shortLabel || label }; }

function Landing() { return <div className="public-landing"><div className="landing-card"><div className="landing-copy"><Typography.Text type="secondary">FRP Tunnel Platform</Typography.Text><Typography.Title>像 mefrp 一样直接好用的内网穿透控制台</Typography.Title><Typography.Paragraph>把套餐、隧道、节点、客户端和测速放在一个清晰的工具面板里，用户只需要完成自己的操作。</Typography.Paragraph><Space><Button type="primary" size="large" onClick={() => location.hash = 'login'}>登录</Button><Button size="large" onClick={() => location.hash = 'register'}>注册</Button></Space></div><div className="landing-side"><Typography.Title style={{ color: '#fff' }} level={3}>开始使用</Typography.Title><div className="quick-flow"><div>开通套餐</div><div>创建隧道</div><div>启动客户端</div><div>访问公网地址</div></div></div></div></div>; }

function renderPage(route, state, load, log, setLog) {
  const props = { state, load, log, setLog };
  return ({ overview: <Overview {...props} />, quickstart: <QuickStart {...props} />, tunnels: <Tunnels {...props} />, create: <CreateTunnel {...props} />, speedtest: <SpeedTest {...props} />, nodes: <Nodes {...props} />, client: <Client {...props} />, domains: <Domains {...props} />, billing: <Billing {...props} />, redeem: <Redeem {...props} />, account: <Account {...props} />, help: <Help /> })[route] || <Overview {...props} />;
}

function Overview({ state }) {
  const sub = state.sub || {};
  const traffic = state.traffic || {};
  const onlineNodes = (state.nodes || []).filter(n => n.status === 'online').length;
  const tunnelCounts = state.tunnels.reduce((acc, t) => { acc[String(t.type || '').toLowerCase()] = (acc[String(t.type || '').toLowerCase()] || 0) + 1; return acc; }, {});
  const trafficLeft = traffic.traffic_left_bytes ?? Math.max(0, (traffic.traffic_limit_bytes || 0) - (traffic.traffic_used_bytes || 0));
  const planBandwidth = subscriptionBandwidth(sub);
  return <div className="panel-stack user-dashboard">
    <section className="user-hero-card">
      <div>
        <Typography.Text className="eyebrow">ME FRP STYLE WORKBENCH</Typography.Text>
        <Typography.Title level={2}>欢迎回来，{state.me?.email || '用户'}</Typography.Title>
        <Typography.Paragraph>这里直接展示你最常用的资源和操作：创建隧道、下载客户端、查看流量、执行测速。</Typography.Paragraph>
        <Space wrap><Button type="primary" size="large" onClick={() => location.hash = 'create'}>创建隧道</Button><Button size="large" onClick={() => location.hash = 'client'}>下载客户端</Button><Button size="large" onClick={() => location.hash = 'speedtest'}>隧道测速</Button></Space>
      </div>
      <div className="hero-plan-card">
        <span>当前套餐</span>
        <strong>{sub.status === 'active' ? sub.plan_name : '未开通'}</strong>
        <p>{sub.status === 'active' ? `到期时间：${formatTime(sub.expires_at)}` : '购买套餐或兑换激活码后即可创建隧道。'}</p>
        <Button block onClick={() => location.hash = sub.status === 'active' ? 'redeem' : 'billing'}>{sub.status === 'active' ? '兑换更多时长' : '去开通套餐'}</Button>
      </div>
    </section>
    <div className="metric-grid"><MetricCard title="剩余流量" value={formatBytes(trafficLeft)} trend={`${formatBytes(traffic.traffic_used_bytes)} 已用`} /><MetricCard title="隧道数量" value={state.tunnels.length} trend={`TCP ${tunnelCounts.tcp || 0} / HTTP ${tunnelCounts.http || 0}`} /><MetricCard title="在线节点" value={onlineNodes} trend={`共 ${state.nodes.length} 个节点`} /><MetricCard title="套餐带宽" value={formatMbps(planBandwidth)} trend={sub.status || 'inactive'} /></div>
    <div className="quick-action-grid">
      <QuickAction title="创建公网入口" desc="填写本地地址和端口，快速生成可访问地址。" action="立即创建" route="create" />
      <QuickAction title="同步本地客户端" desc="下载客户端后同步配置，让隧道上线。" action="查看客户端" route="client" />
      <QuickAction title="查看节点" desc="选择延迟更低、状态在线的节点。" action="节点状态" route="nodes" />
      <QuickAction title="执行测速" desc="测试下载、上传和链路连通质量。" action="开始测速" route="speedtest" />
    </div>
    <div className="two-column"><Card title="最近隧道" bordered={false} extra={<Button type="link" onClick={() => location.hash = 'tunnels'}>全部隧道</Button>}>{state.tunnels.length ? <TunnelTable data={state.tunnels.slice(0, 6)} /> : <Empty description="还没有隧道，先创建一个公网入口" />}</Card><Card title="使用提醒" bordered={false}><div className="notice-list"><div><b>先创建隧道</b><span>选择协议、本地端口和节点。</span></div><div><b>再启动客户端</b><span>客户端会把本地服务接到公网入口。</span></div><div><b>最后测试访问</b><span>用公网地址或测速工具确认可用。</span></div></div></Card></div>
  </div>;
}
function QuickAction({ title, desc, action, route }) { return <Card bordered={false} className="quick-action-card" onClick={() => location.hash = route}><Typography.Title level={4}>{title}</Typography.Title><Typography.Paragraph>{desc}</Typography.Paragraph><Button type="link">{action}</Button></Card>; }
function QuickStart({ state }) { return <Card bordered={false} className="page-card"><Steps direction="vertical" current={state.sub?.status === 'active' ? 2 : 0} items={[{ title: '开通套餐', description: '购买套餐或输入兑换码激活权益' }, { title: '创建隧道', description: '填写协议、本地地址、端口和节点' }, { title: '启动客户端', description: '下载客户端并同步配置' }, { title: '访问公网地址', description: '在隧道列表复制公网入口并测试访问' }]} /></Card>; }
function TunnelTable({ data }) { return <Table size="middle" rowKey="id" dataSource={data} pagination={data.length > 8 ? { pageSize: 8 } : false} columns={[{ title: 'ID', dataIndex: 'id', width: 70, render: v => <Typography.Text type="secondary">#{v}</Typography.Text> }, { title: '名称', dataIndex: 'name' }, { title: '协议', dataIndex: 'type', render: v => <Tag color="blue">{v}</Tag> }, { title: '公网入口', dataIndex: 'public_url', render: v => v ? <CodeLine text={v} /> : '-' }, { title: '本地服务', render: (_, r) => `${r.local_host}:${r.local_port}` }, { title: '状态', dataIndex: 'status', render: v => <StatusBadge status={v} /> }]} />; }
function Tunnels({ state }) { const [open, setOpen] = useState(null); return <Card bordered={false} className="page-card"><div className="table-toolbar"><Alert type="info" showIcon message="表格 + Drawer 动线" description="点击行可查看隧道详情和客户端配置要点。" /><Button type="primary" onClick={() => location.hash = 'create'}>创建隧道</Button></div><Table rowKey="id" dataSource={state.tunnels} onRow={r => ({ onClick: () => setOpen(r) })} columns={[{ title: 'ID', dataIndex: 'id', width: 70 }, { title: '名称', dataIndex: 'name' }, { title: '协议', dataIndex: 'type', render: v => <Tag color="blue">{v}</Tag> }, { title: '公网入口', dataIndex: 'public_url', render: v => v ? <CodeLine text={v} /> : '-' }, { title: '限速', dataIndex: 'effective_bandwidth_limit_kbps', render: formatMbps }, { title: '创建时间', dataIndex: 'created_at', render: formatTime }, { title: '状态', dataIndex: 'status', render: v => <StatusBadge status={v} /> }]} /><Drawer open={Boolean(open)} onClose={() => setOpen(null)} title="隧道详情" width={520}>{open && <Descriptions column={1} bordered items={Object.entries(open).map(([k, v]) => ({ key: k, label: k, children: typeof v === 'object' ? JSON.stringify(v) : String(v ?? '-') }))} />}</Drawer></Card>; }

const protocolMeta = {
  tcp: { title: 'TCP', desc: '远程桌面、SSH、数据库等 TCP 服务', placeholder: '例如 SSH / RDP / MySQL' },
  udp: { title: 'UDP', desc: '游戏、语音、实时传输等 UDP 服务', placeholder: '例如 Game / Voice' },
  http: { title: 'HTTP', desc: '普通网站、Webhook、面板服务', placeholder: 'app.example.com' },
  https: { title: 'HTTPS', desc: '需要安全访问的网站服务', placeholder: 'secure.example.com' },
};
function CreateTunnel({ state, load }) {
  const [form] = Form.useForm();
  const [step, setStep] = useState(0);
  const [protocol, setProtocol] = useState('tcp');
  const sub = state.sub || {};
  async function next() {
    try {
      await form.validateFields(step === 0 ? ['name', 'local_host', 'local_port'] : []);
      setStep(1);
    } catch {}
  }
  async function submit() {
    try {
      const values = await form.validateFields();
      const payload = { ...values, type: protocol, node_id: values.node_id || 0, bandwidth_limit_kbps: values.bandwidth_limit_kbps || 0 };
      if (!['http','https'].includes(protocol)) delete payload.domain;
      const t = await api.post('/api/tunnels', payload);
      message.success('隧道已创建');
      await load();
      Modal.success({ title: '创建成功', content: <div><p>公网入口：</p><CodeLine text={t.public_url || `${t.remote_port || ''}`} /></div> });
      location.hash = 'tunnels';
    } catch (err) { if (err?.errorFields) return; message.error(err.message); }
  }
  return <div className="create-layout">
    <Card bordered={false} className="page-card create-main-card" title="创建隧道" extra={<Tag color="blue">{protocol.toUpperCase()}</Tag>}>
      <Form layout="vertical" form={form} initialValues={{ local_host: '127.0.0.1', local_port: 80 }}>
        <div className="protocol-grid mefrp-protocol-grid">{Object.entries(protocolMeta).map(([key, meta]) => <button type="button" key={key} className={`protocol-card ${protocol===key?'active':''}`} onClick={() => setProtocol(key)}><strong>{meta.title}</strong><span>{meta.desc}</span><em>{protocolEnabled(sub, key) ? '已开通' : '需套餐支持'}</em></button>)}</div>
        {step === 0 && <div className="form-section"><Typography.Title level={4}>基础信息</Typography.Title><Row gutter={16}><Col xs={24} md={12}><Form.Item name="name" label="隧道名称" rules={[{ required: true, message: '请输入隧道名称' }]}><Input placeholder={protocolMeta[protocol].placeholder} /></Form.Item></Col><Col xs={24} md={12}><Form.Item name="node_id" label="节点"><Select allowClear placeholder="自动选择或指定节点" options={(state.nodes || []).map(n => ({ value: n.id, label: `${n.name} / ${n.status || 'unknown'}` }))} /></Form.Item></Col></Row><Row gutter={16}><Col xs={24} md={12}><Form.Item name="local_host" label="本地地址" rules={[{ required: true, message: '请输入本地地址' }]}><Input placeholder="127.0.0.1" /></Form.Item></Col><Col xs={24} md={12}><Form.Item name="local_port" label="本地端口" rules={[{ required: true, message: '请输入本地端口' }]}><InputNumber min={1} max={65535} style={{ width: '100%' }} /></Form.Item></Col></Row></div>}
        {step === 1 && <div className="form-section"><Typography.Title level={4}>入口与高级设置</Typography.Title>{['http','https'].includes(protocol) && <Form.Item name="domain" label="绑定域名" rules={[{ required: true, message: 'HTTP/HTTPS 隧道需要填写域名' }]}><Input placeholder={protocolMeta[protocol].placeholder} /></Form.Item>}<Form.Item name="bandwidth_limit_kbps" label="单隧道限速（可选）" extra="不填写则使用套餐带宽。"><InputNumber min={0} max={subscriptionBandwidth(sub) || undefined} addonAfter="Kbps" style={{ width: '100%' }} /></Form.Item><Descriptions bordered size="small" column={1} items={[{ label: '协议', children: protocol.toUpperCase() }, { label: '本地服务', children: `${form.getFieldValue('local_host') || '127.0.0.1'}:${form.getFieldValue('local_port') || '-'}` }, { label: '套餐带宽', children: formatMbps(subscriptionBandwidth(sub)) }]} /></div>}
        <div className="create-footer"><Button onClick={() => step === 0 ? location.hash = 'tunnels' : setStep(0)}>{step === 0 ? '返回列表' : '上一步'}</Button>{step === 0 ? <Button type="primary" onClick={next}>下一步</Button> : <Button type="primary" onClick={submit}>创建隧道</Button>}</div>
      </Form>
    </Card>
    <Card bordered={false} className="page-card create-side-card" title="创建提示"><div className="notice-list"><div><b>先选协议</b><span>TCP/UDP 会分配端口，HTTP/HTTPS 需要域名。</span></div><div><b>填本地服务</b><span>例如本机网站通常是 127.0.0.1:80。</span></div><div><b>启动客户端</b><span>创建后到客户端页同步配置。</span></div></div></Card>
  </div>;
}

function Nodes({ state }) { return <div className="card-grid">{state.nodes.map(n => <Card key={n.id} bordered={false} className="page-card" title={n.name} extra={<StatusBadge status={n.status} />}><Descriptions column={1} size="small" items={[{ label: '入口域名', children: n.frp_entry_domain || '-' }, { label: 'frps', children: `${n.server_addr || '-'}:${n.frp_server_port || '-'}` }, { label: 'TCP', children: `${n.tcp_port_start}-${n.tcp_port_end}` }, { label: 'UDP', children: `${n.udp_port_start}-${n.udp_port_end}` }, { label: '最后在线', children: formatTime(n.last_seen_at) }]} /></Card>)}</div>; }
function Client({ state }) { const token = api.token(); return <div className="two-column"><Card title="客户端下载" bordered={false}>{(state.topology?.downloads || []).map(d => <Card key={d.platform} size="small" style={{ marginBottom: 12 }}><Space direction="vertical"><Typography.Title level={5}>{d.label}</Typography.Title><a href={d.url}>{d.url}</a></Space></Card>)}</Card><Card title="连接信息" bordered={false}><Descriptions column={1} bordered items={[{ label: 'API Server', children: <CodeLine text={apiOrigin()} /> }, { label: 'Token', children: <Space><Typography.Text code ellipsis style={{ maxWidth: 210 }}>{token}</Typography.Text><CopyText text={token}>Token</CopyText></Space> }, { label: '服务节点', children: `${state.clientConfig?.server_addr || '-'}:${state.clientConfig?.server_port || '-'}` }]} /></Card></div>; }
function Domains() { const [form] = Form.useForm(); async function submit(v){ try { const res = await api.post('/api/user/certificates/request', v); Modal.info({ title: '申请结果', content: <pre>{JSON.stringify(res, null, 2)}</pre> }); } catch(e){ message.error(e.message); } } return <Card bordered={false} className="page-card"><Alert showIcon type="info" message="域名证书按需保留" description="用户提交申请，后台可统一检查 CNAME、生成 Nginx 配置和续期。" style={{ marginBottom: 16 }} /><Form form={form} layout="vertical" onFinish={submit}><Form.Item name="domain" label="域名" rules={[{ required: true }]}><Input placeholder="app.example.com" /></Form.Item><Form.Item name="email" label="证书邮箱"><Input /></Form.Item><Button type="primary" htmlType="submit">申请证书</Button></Form></Card>; }
function Billing({ state, load }) { const [payType, setPayType] = useState('wxpay'); const [pay, setPay] = useState(null); async function buy(plan){ try { const order = await api.post('/api/payments/epay/orders', { plan_id: plan.id, pay_type: payType }); setPay(order); message.success('订单已创建'); await load(); } catch(e){ message.error(e.message); } } return <div className="panel-stack"><Alert showIcon type="info" message="当前支付方式" description="微信支付 -> pay_type=wxpay -> 通道 wxpay_zg。" /><Radio.Group value={payType} onChange={e => setPayType(e.target.value)}><Radio.Button value="wxpay">微信支付</Radio.Button><Radio.Button value="alipay">支付宝</Radio.Button></Radio.Group><div className="plan-grid">{state.plans.map(p => <Card className="plan-card" key={p.id} title={p.name} bordered={false} extra={<Tag>{formatMoney(p.price_cents)}</Tag>}><Space direction="vertical" style={{ width: '100%' }}><Typography.Paragraph>{p.description}</Typography.Paragraph><Descriptions size="small" column={1} items={[{ label: '时长', children: `${p.duration_days} 天` }, { label: '流量', children: formatBytes(p.traffic_limit_bytes) }, { label: '带宽', children: formatMbps(p.bandwidth_limit_kbps) }, { label: '隧道', children: p.max_tunnels }]} /><Button type="primary" block onClick={() => buy(p)}>购买</Button></Space></Card>)}</div>{pay && <Card title="支付订单" bordered={false}><Descriptions column={1} bordered items={[{ label: '订单号', children: pay.out_trade_no }, { label: '金额', children: pay.money }, { label: '方式', children: pay.pay_type }, { label: '支付链接', children: <a target="_blank" href={pay.pay_url}>{pay.pay_url}</a> }]} /></Card>}</div>; }
function Redeem({ load }) { const [form] = Form.useForm(); const [result, setResult] = useState(null); async function submit(v){ try { const res = await api.post('/api/user/redeem', v); setResult(res); message.success('兑换成功'); await load(); } catch(e){ message.error(e.message); } } return <Card bordered={false} className="page-card"><Alert showIcon type="info" message="兑换码已绑定套餐" style={{ marginBottom: 16 }} /><Form form={form} layout="inline" onFinish={submit}><Form.Item name="code" rules={[{ required: true }]}><Input placeholder="DEMO-PLAN-2026" /></Form.Item><Button type="primary" htmlType="submit">兑换</Button></Form>{result && <Card style={{ marginTop: 16 }}><Descriptions column={1} bordered items={[{ label: '套餐', children: result.plan_name }, { label: '到期', children: formatTime(result.expires_at) }, { label: '状态', children: <StatusBadge status={result.status} /> }]} /></Card>}</Card>; }
function SpeedTest({ state, log, setLog, load }) {
  const [form] = Form.useForm();
  const [current, setCurrent] = useState(0);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState(null);
  async function local(base, path, body, localToken){ const headers = { 'Content-Type': 'application/json' }; if (localToken) headers['X-Local-Token'] = localToken; const res = await fetch(base.replace(/\/$/,'') + path, { method: 'POST', headers, body: JSON.stringify(body || {}) }); const json = await res.json(); if(!res.ok || json.success === false) throw new Error(json.message || res.statusText); return json.data; }
  async function run(v){ setRunning(true); setResult(null); setLog('开始测速'); try { localStorage.setItem('localClientBase', v.local_client || ''); localStorage.setItem('localClientToken', v.local_token || ''); setCurrent(0); const bench = await local(v.local_client, '/api/speed-tests/prepare', { type: v.type }, v.local_token); setLog(l => l + '\n本地测速服务已准备：' + JSON.stringify(bench)); setCurrent(1); const tunnel = await api.post('/api/speed-tests/tunnels', { type: v.type, local_host: bench.host, local_port: bench.port, node_id: v.node_id || 0 }); setLog(l => l + '\n临时入口已创建：' + (tunnel.public_url || tunnel.id)); setCurrent(2); await local(v.local_client, '/api/config/sync', { api_base: apiOrigin(), token: api.token(), speed_test_id: tunnel.id }, v.local_token); await local(v.local_client, '/api/frpc/restart', {}, v.local_token); setCurrent(3); const data = await api.post(`/api/speed-tests/${tunnel.id}/run`, { download_bytes: (v.download_mb || 4) * 1024 * 1024, upload_bytes: (v.upload_mb || 2) * 1024 * 1024, duration_seconds: v.duration_seconds || 45 }); setResult(data); setCurrent(4); setLog(l => l + '\n测速完成：' + JSON.stringify(data, null, 2)); await local(v.local_client, '/api/speed-tests/cleanup', {}, v.local_token).catch(()=>{}); await load(); } catch(e){ message.error(e.message); setLog(l => l + '\nERROR ' + e.message); } finally { setRunning(false); } }
  const metrics = result?.metrics || {};
  return <div className="speed-layout"><Card bordered={false} className="page-card" title="本地客户端连接"><Form layout="vertical" form={form} onFinish={run} initialValues={{ local_client: localStorage.getItem('localClientBase') || 'http://127.0.0.1:18080', local_token: localStorage.getItem('localClientToken') || '', type: 'tcp', download_mb: 4, upload_mb: 2, duration_seconds: 45 }}><Form.Item name="local_client" label="本地客户端 API" rules={[{ required: true }]}><Input onChange={e => localStorage.setItem('localClientBase', e.target.value)} /></Form.Item><Form.Item name="local_token" label="本地访问密钥" rules={[{ required: true, message: '请输入本地客户端访问密钥' }]}><Input.Password placeholder="从本地客户端页面获取" onChange={e => localStorage.setItem('localClientToken', e.target.value)} /></Form.Item><Row gutter={12}><Col xs={24} md={12}><Form.Item name="node_id" label="节点"><Select allowClear placeholder="自动选择" options={state.nodes.map(n => ({ value: n.id, label: n.name }))} /></Form.Item></Col><Col xs={24} md={12}><Form.Item name="type" label="协议"><Select options={['tcp','udp','http','https'].map(v => ({ value: v, label: v.toUpperCase() }))}/></Form.Item></Col></Row><Row gutter={12}><Col xs={24} md={8}><Form.Item name="download_mb" label="下载 MB"><InputNumber min={1} style={{ width:'100%' }} /></Form.Item></Col><Col xs={24} md={8}><Form.Item name="upload_mb" label="上传 MB"><InputNumber min={1} style={{ width:'100%' }} /></Form.Item></Col><Col xs={24} md={8}><Form.Item name="duration_seconds" label="超时秒数"><InputNumber min={10} style={{ width:'100%' }} /></Form.Item></Col></Row><Button type="primary" htmlType="submit" loading={running} block>开始测速</Button></Form></Card><div className="panel-stack"><Card bordered={false} className="page-card" title="测速进度"><Steps className="speed-steps" current={current} direction="vertical" items={['准备本地测速服务','创建临时公网入口','同步并重启客户端','执行下载/上传测速','清理临时资源'].map(title => ({ title }))} /></Card>{result && <div className="speed-result-grid"><MetricCard title="下载速度" value={formatMbps(metrics.download_average_kbps)} trend="平均值" /><MetricCard title="上传速度" value={formatMbps(metrics.upload_average_kbps)} trend="平均值" /><MetricCard title="状态" value={result.finished ? '完成' : '进行中'} trend={result.public_url || '-'} /></div>}<LogPanel title="测速日志" value={log} /></div></div>;
}
function Account({ state }) { const sub = state.sub || {}; const traffic = state.traffic || {}; return <div className="two-column"><Card title="账户" bordered={false}><Descriptions column={1} bordered items={[{ label: 'Email', children: state.me?.email }, { label: 'Token', children: <Space><Typography.Text code ellipsis style={{ maxWidth: 220 }}>{api.token()}</Typography.Text><CopyText text={api.token()} /></Space> }, { label: '状态', children: <StatusBadge status={state.me?.status} /> }]} /></Card><Card title="套餐" bordered={false}><Progress percent={percent(traffic.traffic_used_bytes, traffic.traffic_limit_bytes)} /><Descriptions column={1} items={[{ label: '套餐', children: sub.plan_name || '-' }, { label: '到期', children: formatTime(sub.expires_at) }, { label: '流量', children: `${formatBytes(traffic.traffic_used_bytes)} / ${formatBytes(traffic.traffic_limit_bytes)}` }]} /></Card></div>; }
function Help() { return <div className="help-grid"><Card title="如何创建隧道"><p>进入“创建隧道”，选择协议，填写本地地址和端口，再点击创建即可。</p></Card><Card title="客户端怎么用"><p>下载本地客户端，登录后同步配置，然后启动客户端让隧道上线。</p></Card><Card title="什么时候填域名"><p>HTTP/HTTPS 隧道需要绑定域名；TCP/UDP 隧道会分配公网端口。</p></Card><Card title="测速失败怎么办"><p>确认本地客户端正在运行，并且本地访问密钥填写正确。</p></Card></div>; }
