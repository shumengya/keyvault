import React, { useState } from 'react';
import './PasswordLogin.css';

const PasswordLogin = ({ onLogin }) => {
  const [first, setFirst] = useState('');
  const [second, setSecond] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    const message = await onLogin(first, second);
    if (message) {
      setError(message);
      setFirst('');
      setSecond('');
    }
    setLoading(false);
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-header">
          <img
            src={`${process.env.PUBLIC_URL}/logo.png`}
            alt="萌芽密码管理器"
            className="login-logo"
          />
          <p>请输入两个访问密码</p>
          <p className="login-hint">两个都输对才能进入，先后顺序不限</p>
        </div>
        <form onSubmit={handleSubmit} className="login-form">
          <div className="form-group">
            <input
              type="password"
              value={first}
              onChange={(e) => setFirst(e.target.value)}
              placeholder="访问密码 1"
              aria-label="访问密码 1"
              className="password-input"
              autoComplete="off"
              required
              autoFocus
            />
          </div>
          <div className="form-group">
            <input
              type="password"
              value={second}
              onChange={(e) => setSecond(e.target.value)}
              placeholder="访问密码 2"
              aria-label="访问密码 2"
              className="password-input"
              autoComplete="off"
              required
            />
          </div>
          {error && (
            <div className="error-message" role="alert">
              {error}
            </div>
          )}
          <button type="submit" className="login-button" disabled={loading}>
            {loading ? '验证中...' : '登录'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default PasswordLogin;
