import React from 'react';
import { createRoot } from 'react-dom/client';
import { ConfigProvider, App as AntApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { antdTheme } from '../../shared/frontend/theme/antdTheme.js';
import App from './App.jsx';
import './styles.css';

createRoot(document.getElementById('root')).render(
  <ConfigProvider locale={zhCN} theme={antdTheme}>
    <AntApp>
      <App />
    </AntApp>
  </ConfigProvider>
);
