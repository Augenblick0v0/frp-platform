import React, { useMemo, useState } from 'react';
import { Layout, Menu, Button, Space, Typography, Tooltip, Breadcrumb, Avatar, Dropdown } from 'antd';
import { MenuFoldOutlined, MenuUnfoldOutlined, ReloadOutlined, LogoutOutlined, UserOutlined } from '@ant-design/icons';
import './global-import.css';

const { Header, Sider, Content } = Layout;
const { Text, Title } = Typography;

export function CollapsedMenuLabel({ text }) {
  const chars = Array.from(String(text || ''));
  if (chars.length === 4 && chars.every(ch => /[一-龥]/.test(ch))) {
    return <span className="collapsed-menu-label two-by-two"><span>{chars.slice(0, 2).join('')}</span><span>{chars.slice(2).join('')}</span></span>;
  }
  if (chars.length === 2 && chars.every(ch => /[一-龥]/.test(ch))) {
    return <span className="collapsed-menu-label vertical"><span>{chars[0]}</span><span>{chars[1]}</span></span>;
  }
  return <span className="collapsed-menu-label icon-only">{chars.slice(0, 2).join('')}</span>;
}

function renderCollapsedItems(items) {
  return items.map(group => ({
    ...group,
    label: group.label,
    children: (group.children || []).map(item => ({
      ...item,
      label: <Tooltip placement="right" title={item.rawLabel || item.label}><CollapsedMenuLabel text={item.shortLabel || item.rawLabel || item.label} /></Tooltip>,
    })),
  }));
}

export function AppShell({
  title,
  subtitle,
  brand = 'FRP Tunnel',
  brandSub = 'Control Plane',
  menuItems = [],
  selectedKey,
  onSelect,
  children,
  userLabel,
  onLogout,
  onRefresh,
  extra,
  storageKey = 'sidebarCollapsed',
}) {
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(storageKey) === 'true');
  const items = useMemo(() => collapsed ? renderCollapsedItems(menuItems) : menuItems, [collapsed, menuItems]);
  const dropdownItems = [{ key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true }];
  const setAndStore = value => { setCollapsed(value); localStorage.setItem(storageKey, String(value)); };
  return (
    <Layout className="app-shell">
      <Sider className="app-sider" width={232} collapsedWidth={72} collapsed={collapsed} trigger={null}>
        <div className="brand-block">
          <div className="brand-logo">F</div>
          {!collapsed && <div><strong>{brand}</strong><small>{brandSub}</small></div>}
        </div>
        <Menu mode="inline" selectedKeys={[selectedKey]} items={items} onClick={e => onSelect?.(e.key)} />
      </Sider>
      <Layout>
        <Header className="app-header">
          <Space size={14}>
            <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => setAndStore(!collapsed)} />
            <div>
              <Breadcrumb items={[{ title: brand }, { title }]} />
              <Title level={4} className="page-title">{title}</Title>
            </div>
          </Space>
          <Space>
            {extra}
            <Button icon={<ReloadOutlined />} onClick={onRefresh}>刷新</Button>
            <Dropdown menu={{ items: dropdownItems, onClick: ({ key }) => key === 'logout' && onLogout?.() }}>
              <Space className="user-chip"><Avatar size="small" icon={<UserOutlined />} /><Text>{userLabel || '已登录'}</Text></Space>
            </Dropdown>
          </Space>
        </Header>
        <Content className="app-content">
          {subtitle && <Text className="page-subtitle">{subtitle}</Text>}
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}
