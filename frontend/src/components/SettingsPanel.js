import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './PasswordForm.css';
import './SettingsPanel.css';
import PasswordChangeForm from './PasswordChangeForm';
import API_BASE from '../config/api';

const SettingsPanel = ({ onClose, onLogout }) => {
  const [hours, setHours] = useState('');
  const [limits, setLimits] = useState({ min: 1, max: 8760 });
  const [message, setMessage] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  useEffect(() => {
    axios
      .get(`${API_BASE}/settings`)
      .then((response) => {
        setHours(String(response.data.sessionTTLHours));
        setLimits({
          min: response.data.minSessionTTLHours,
          max: response.data.maxSessionTTLHours,
        });
      })
      .catch((error) => {
        setMessage({ type: 'error', text: error.response?.data?.error || '读取设置失败' });
      });
  }, []);

  const handleSave = async (e) => {
    e.preventDefault();
    setMessage(null);
    setSaving(true);
    try {
      await axios.put(`${API_BASE}/settings/session`, { hours: Number(hours) });
      setMessage({ type: 'success', text: '已保存，对之后的新登录生效' });
    } catch (error) {
      setMessage({ type: 'error', text: error.response?.data?.error || '保存失败' });
    } finally {
      setSaving(false);
    }
  };

  const hoursNumber = Number(hours);
  const daysHint = hoursNumber > 0 && hoursNumber % 24 === 0 ? `（${hoursNumber / 24} 天）` : '';

  return (
    <div className="form-overlay" onClick={onClose}>
      <div
        className="form-modal settings-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="form-header">
          <h2 id="settings-title">设置</h2>
          <button type="button" className="close-button" onClick={onClose} aria-label="关闭设置">
            ✕
          </button>
        </div>

        <div className="settings-body">
          <section className="settings-section">
            <h3>登录有效期</h3>
            <form onSubmit={handleSave}>
              <label htmlFor="session-hours">有效期（小时）{daysHint}</label>
              <div className="settings-inline">
                <input
                  id="session-hours"
                  type="number"
                  min={limits.min}
                  max={limits.max}
                  step="1"
                  value={hours}
                  onChange={(e) => setHours(e.target.value)}
                  required
                />
                <button type="submit" className="settings-button" disabled={saving}>
                  {saving ? '保存中...' : '保存'}
                </button>
              </div>
              <p className="settings-hint">
                登录后超过这个时间，需要重新输入两个密码。修改只对之后的新登录生效，已登录的设备保持原来的到期时间。
              </p>
              {message && (
                <p
                  role={message.type === 'error' ? 'alert' : 'status'}
                  className={`settings-message settings-message--${message.type}`}
                >
                  {message.text}
                </p>
              )}
            </form>
          </section>

          <PasswordChangeForm />

          <section className="settings-section">
            <h3>退出登录</h3>
            <button
              type="button"
              className="settings-button settings-button--danger"
              onClick={onLogout}
            >
              退出登录
            </button>
          </section>
        </div>
      </div>
    </div>
  );
};

export default SettingsPanel;
