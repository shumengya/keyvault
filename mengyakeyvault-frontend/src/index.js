import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';
import * as serviceWorkerRegistration from './serviceWorkerRegistration';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

// 注册 Service Worker，启用 PWA 离线缓存能力
serviceWorkerRegistration.register({
  onSuccess: () => {
    console.log('[PWA] 应用已缓存，可离线使用');
  },
  onUpdate: (registration) => {
    // 发现新版本时，提示用户刷新
    if (window.confirm('🌱 萌芽密码管理器有新版本，是否立即刷新？')) {
      if (registration && registration.waiting) {
        registration.waiting.postMessage({ type: 'SKIP_WAITING' });
      }
      window.location.reload();
    }
  },
});
