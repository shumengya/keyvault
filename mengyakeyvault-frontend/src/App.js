import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';
import PasswordLogin from './components/PasswordLogin';
import PasswordManager from './components/PasswordManager';

// API 地址配置：优先使用环境变量，否则根据构建模式自动选择
const API_BASE = process.env.REACT_APP_API_BASE || 
  (process.env.NODE_ENV === 'production' 
    ? 'https://keyvault.api.shumengya.top/api' 
    : 'http://localhost:8080/api');
const STORAGE_KEY = 'mengyakeyvault_authenticated';

function App() {
  const [authenticated, setAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // 检查是否已认证
    const cached = localStorage.getItem(STORAGE_KEY);
    if (cached === 'true') {
      setAuthenticated(true);
    }
    setLoading(false);
  }, []);

  const handleLogin = async (password) => {
    try {
      const response = await axios.post(`${API_BASE}/verify`, { password });
      if (response.data.success) {
        localStorage.setItem(STORAGE_KEY, 'true');
        setAuthenticated(true);
        return true;
      }
      return false;
    } catch (error) {
      console.error('登录失败:', error);
      return false;
    }
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

  return <PasswordManager />;
}

export default App;
