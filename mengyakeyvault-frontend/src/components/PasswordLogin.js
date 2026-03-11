import React, { useState } from 'react';
import './PasswordLogin.css';

const PasswordLogin = ({ onLogin }) => {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    const success = await onLogin(password);
    if (!success) {
      setError('密码错误，请重试');
      setPassword('');
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
          <p>请输入访问密码</p>
        </div>
        <form onSubmit={handleSubmit} className="login-form">
          <div className="form-group">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="请输入密码"
              className="password-input"
              required
              autoFocus
            />
          </div>
          {error && <div className="error-message">{error}</div>}
          <button type="submit" className="login-button" disabled={loading}>
            {loading ? '验证中...' : '登录'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default PasswordLogin;
