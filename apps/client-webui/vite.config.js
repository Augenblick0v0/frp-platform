import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      react: path.resolve(__dirname, 'node_modules/react'),
      'react-dom': path.resolve(__dirname, 'node_modules/react-dom'),
      'react/jsx-runtime': path.resolve(__dirname, 'node_modules/react/jsx-runtime.js'),
      antd: path.resolve(__dirname, 'node_modules/antd'),
      '@ant-design/icons': path.resolve(__dirname, 'node_modules/@ant-design/icons'),
    },
    dedupe: ['react', 'react-dom', 'antd', '@ant-design/icons'],
  },
  server: { port: 5175, proxy: { '/api': 'http://127.0.0.1:8080' } },
  build: { outDir: 'dist', emptyOutDir: true, rollupOptions: { output: { manualChunks: { vendor: ['react', 'react-dom', 'antd', '@ant-design/icons'] } } } }
});
