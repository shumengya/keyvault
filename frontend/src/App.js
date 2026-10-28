import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';
import PasswordLogin from './components/PasswordLogin';
import PasswordManager from './components/PasswordManager';
import API_BASE from './config/api';

// 旧版本把"已登录"标记存在 localStorage,现在改由服务端会话判断,顺手清掉
const LEGACY_STORAGE_KEY = 'mengyakeyvault_authenticated';

function App() {
  const [authenticated, setAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    localStorage.removeItem(LEGACY_STORAGE_KEY);

    // 任何接口返回 401 都说明登录已过期,回到登录页。
    // 登录接口自己的 401 是"密码错误",不能算过期。
    const interceptorId = axios.interceptors.response.use(
      (response) => response,
      (error) => {
        const url = error.config?.url || '';
        if (error.response?.status === 401 && !url.endsWith('/login')) {
          setAuthenticated(false);
        }
        return Promise.reject(error);
      }
    );

    axios
      .get(`${API_BASE}/session`)
      .then((response) => setAuthenticated(Boolean(response.data.authenticated)))
      .catch((error) => console.error('检查登录状态失败:', error))
      .finally(() => setLoading(false));

    return () => axios.interceptors.response.eject(interceptorId);
  }, []);

  // 成功返回空串,失败返回要展示给用户的错误信息
  const handleLogin = async (first, second) => {
    try {
      await axios.post(`${API_BASE}/login`, { first, second });
      setAuthenticated(true);
      return '';
    } catch (error) {
      return error.response?.data?.error || '无法连接到服务器';
    }
  };

  const handleLogout = async () => {
    try {
      await axios.post(`${API_BASE}/logout`);
    } catch (error) {
      // 服务端没退成功就不能假装已退出,否则刷新页面又登录回去了
      console.error('退出登录失败:', error);
      alert('退出登录失败，请重试');
      return;
    }
    setAuthenticated(false);
  };

  if (loading) {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
      </div>
    );
  }

  if (!authenticated) {
    return <PasswordLogin onLogin={handleLogin} />;
  }

  return <PasswordManager onLogout={handleLogout} />;
}

export default App;
