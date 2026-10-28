import React, { useState } from 'react';
import axios from 'axios';
import API_BASE from '../config/api';

const EMPTY_FIELDS = {
  currentFirst: '',
  currentSecond: '',
  newFirst: '',
  newSecond: '',
  confirmFirst: '',
  confirmSecond: '',
};

const GROUPS = [
  { title: '当前密码', fields: [['currentFirst', '密码 1'], ['currentSecond', '密码 2']] },
  { title: '新密码', fields: [['newFirst', '新密码 1'], ['newSecond', '新密码 2']] },
  { title: '再输一遍新密码', fields: [['confirmFirst', '新密码 1'], ['confirmSecond', '新密码 2']] },
];

const PasswordChangeForm = () => {
  const [fields, setFields] = useState(EMPTY_FIELDS);
  const [message, setMessage] = useState(null);
  const [saving, setSaving] = useState(false);

  const handleChange = (name) => (e) => {
    const value = e.target.value;
    setFields((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setMessage(null);

    // 新密码输错就进不来了，所以要求再输一遍核对
    if (fields.newFirst !== fields.confirmFirst || fields.newSecond !== fields.confirmSecond) {
      setMessage({ type: 'error', text: '两次输入的新密码不一致' });
      return;
    }

    setSaving(true);
    try {
      await axios.put(`${API_BASE}/settings/passwords`, {
        currentFirst: fields.currentFirst,
        currentSecond: fields.currentSecond,
        newFirst: fields.newFirst,
        newSecond: fields.newSecond,
      });
      setFields(EMPTY_FIELDS);
      setMessage({ type: 'success', text: '密码已修改，其他设备需要用新密码重新登录' });
    } catch (error) {
      setMessage({ type: 'error', text: error.response?.data?.error || '修改失败' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="settings-section">
      <h3>修改访问密码</h3>
      <form onSubmit={handleSubmit}>
        {GROUPS.map((group) => (
          <fieldset key={group.title} className="settings-fieldset">
            <legend>{group.title}</legend>
            <div className="settings-grid">
              {group.fields.map(([name, label]) => (
                <div key={name}>
                  <label htmlFor={`password-${name}`}>{label}</label>
                  <input
                    id={`password-${name}`}
                    type="password"
                    value={fields[name]}
                    onChange={handleChange(name)}
                    autoComplete="off"
                    required
                  />
                </div>
              ))}
            </div>
          </fieldset>
        ))}
        <p className="settings-hint">
          登录时必须同时输入这两个密码，顺序不限。两个密码不能相同，每个最长 72 字节（一个汉字占 3 字节）。
        </p>
        {message && (
          <p
            role={message.type === 'error' ? 'alert' : 'status'}
            className={`settings-message settings-message--${message.type}`}
          >
            {message.text}
          </p>
        )}
        <button type="submit" className="settings-button settings-submit" disabled={saving}>
          {saving ? '保存中...' : '修改密码'}
        </button>
      </form>
    </section>
  );
};

export default PasswordChangeForm;
