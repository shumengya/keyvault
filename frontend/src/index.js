import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';

// 已移除 PWA:主动注销旧版本留下的 Service Worker 和缓存。
// 旧页面本身若被缓存住,这段代码跑不到,那种情况靠 public/service-worker.js 自注销兜底。
if ('serviceWorker' in navigator) {
  navigator.serviceWorker
    .getRegistrations()
    .then((registrations) => registrations.forEach((registration) => registration.unregister()));
}
if ('caches' in window) {
  caches.keys().then((names) => names.forEach((name) => caches.delete(name)));
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
