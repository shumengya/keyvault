import React, { useState, useEffect } from 'react';
import './PasswordForm.css';

const PasswordForm = ({ entry, onSave, onCancel }) => {
  const [formData, setFormData] = useState({
    accountType: '网站',
    account: '',
    password: '',
    username: '',
    phone: '',
    email: '',
    website: '',
    officialName: '',
    tags: '',
    logo: '',
  });

  const [passwordOptions, setPasswordOptions] = useState({
    length: 16,
    includeUppercase: true,
    includeLowercase: true,
    includeNumbers: true,
    includeSpecial: true,
  });

  const [showPasswordGenerator, setShowPasswordGenerator] = useState(false);

  useEffect(() => {
    if (entry) {
      setFormData({
        accountType: entry.accountType || '网站',
        account: entry.account || '',
        password: entry.password || '',
        username: entry.username || '',
        phone: entry.phone || '',
        email: entry.email || '',
        website: entry.website || '',
        officialName: entry.officialName || entry.software || '',
        tags: entry.tags || '',
        logo: entry.logo || '',
      });
      setShowPasswordGenerator(false);
    } else {
      // 新建时默认显示密码生成器
      setShowPasswordGenerator(true);
    }
  }, [entry]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onSave(formData);
  };

  const generatePassword = () => {
    const uppercase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
    const lowercase = 'abcdefghijklmnopqrstuvwxyz';
    const numbers = '0123456789';
    const special = '!@#$%^&*()_+-=[]{}|;:,.<>?';

    let charset = '';
    if (passwordOptions.includeLowercase) charset += lowercase;
    if (passwordOptions.includeUppercase) charset += uppercase;
    if (passwordOptions.includeNumbers) charset += numbers;
    if (passwordOptions.includeSpecial) charset += special;

    if (charset === '') {
      alert('请至少选择一种字符类型');
      return;
    }

    let password = '';
    const length = Math.max(4, Math.min(128, passwordOptions.length));

    // 确保至少包含每种选中的字符类型
    if (passwordOptions.includeLowercase) {
      password += lowercase[Math.floor(Math.random() * lowercase.length)];
    }
    if (passwordOptions.includeUppercase) {
      password += uppercase[Math.floor(Math.random() * uppercase.length)];
    }
    if (passwordOptions.includeNumbers) {
      password += numbers[Math.floor(Math.random() * numbers.length)];
    }
    if (passwordOptions.includeSpecial) {
      password += special[Math.floor(Math.random() * special.length)];
    }

    // 填充剩余长度
    for (let i = password.length; i < length; i++) {
      password += charset[Math.floor(Math.random() * charset.length)];
    }

    // 打乱顺序（Fisher-Yates 洗牌算法）
    const passwordArray = password.split('');
    for (let i = passwordArray.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [passwordArray[i], passwordArray[j]] = [passwordArray[j], passwordArray[i]];
    }
    password = passwordArray.join('');

    setFormData((prev) => ({
      ...prev,
      password: password,
    }));
  };

  return (
    <div className="form-overlay" onClick={onCancel}>
      <div className="form-modal" onClick={(e) => e.stopPropagation()}>
        <div className="form-header">
          <h2>{entry ? '编辑密码' : '添加密码'}</h2>
          <button className="close-button" onClick={onCancel}>
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit} className="password-form">
          <div className="form-row">
            <div className="form-group">
              <label>官方名称 *</label>
              <input
                type="text"
                name="officialName"
                value={formData.officialName}
                onChange={handleChange}
                placeholder="例如：MiniMax、GitHub"
                required
              />
            </div>
            <div className="form-group">
              <label>账号类型 *</label>
              <select
                name="accountType"
                value={formData.accountType}
                onChange={handleChange}
                className="form-select"
                required
              >
                <option value="网站">网站</option>
                <option value="软件">软件</option>
              </select>
            </div>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>账号</label>
              <input
                type="text"
                name="account"
                value={formData.account}
                onChange={handleChange}
                placeholder="账号"
              />
            </div>
            <div className="form-group">
              <label>密码</label>
              <div className="password-input-group">
                <input
                  type="text"
                  name="password"
                  value={formData.password}
                  onChange={handleChange}
                  placeholder="密码"
                />
                <button
                  type="button"
                  className="generate-password-btn"
                  onClick={() => {
                    if (!showPasswordGenerator) {
                      setShowPasswordGenerator(true);
                    }
                    generatePassword();
                  }}
                  title="生成随机密码"
                >
                  🎲
                </button>
                <button
                  type="button"
                  className="password-options-btn"
                  onClick={() => setShowPasswordGenerator(!showPasswordGenerator)}
                  title="密码生成选项"
                >
                  ⚙️
                </button>
              </div>
              {showPasswordGenerator && (
                <div className="password-generator-options">
                  <div className="option-row">
                    <label className="option-label">长度：</label>
                    <input
                      type="number"
                      min="4"
                      max="128"
                      value={passwordOptions.length}
                      onChange={(e) =>
                        setPasswordOptions({
                          ...passwordOptions,
                          length: parseInt(e.target.value) || 16,
                        })
                      }
                      className="length-input"
                    />
                  </div>
                  <div className="option-checkboxes">
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={passwordOptions.includeUppercase}
                        onChange={(e) =>
                          setPasswordOptions({
                            ...passwordOptions,
                            includeUppercase: e.target.checked,
                          })
                        }
                      />
                      <span>大写字母 (A-Z)</span>
                    </label>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={passwordOptions.includeLowercase}
                        onChange={(e) =>
                          setPasswordOptions({
                            ...passwordOptions,
                            includeLowercase: e.target.checked,
                          })
                        }
                      />
                      <span>小写字母 (a-z)</span>
                    </label>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={passwordOptions.includeNumbers}
                        onChange={(e) =>
                          setPasswordOptions({
                            ...passwordOptions,
                            includeNumbers: e.target.checked,
                          })
                        }
                      />
                      <span>数字 (0-9)</span>
                    </label>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={passwordOptions.includeSpecial}
                        onChange={(e) =>
                          setPasswordOptions({
                            ...passwordOptions,
                            includeSpecial: e.target.checked,
                          })
                        }
                      />
                      <span>特殊字符 (!@#$...)</span>
                    </label>
                  </div>
                  <button
                    type="button"
                    className="quick-generate-btn"
                    onClick={generatePassword}
                  >
                    🔄 重新生成
                  </button>
                </div>
              )}
            </div>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>用户名</label>
              <input
                type="text"
                name="username"
                value={formData.username}
                onChange={handleChange}
                placeholder="用户名"
              />
            </div>
            <div className="form-group">
              <label>手机号</label>
              <input
                type="tel"
                name="phone"
                value={formData.phone}
                onChange={handleChange}
                placeholder="手机号"
              />
            </div>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>邮箱</label>
              <input
                type="email"
                name="email"
                value={formData.email}
                onChange={handleChange}
                placeholder="邮箱"
              />
            </div>
            <div className="form-group">
              <label>网站地址</label>
              <input
                type="url"
                name="website"
                value={formData.website}
                onChange={handleChange}
                placeholder="https://example.com"
              />
            </div>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label>标签</label>
              <input
                type="text"
                name="tags"
                value={formData.tags}
                onChange={handleChange}
                placeholder="标签（用空格分隔）"
              />
            </div>
            <div className="form-group">
              <label>Logo图标URL（可选）</label>
              <input
                type="url"
                name="logo"
                value={formData.logo}
                onChange={handleChange}
                placeholder="https://example.com/logo.png（留空则自动获取）"
              />
            </div>
          </div>

          <div className="form-actions">
            <button type="button" className="cancel-button" onClick={onCancel}>
              取消
            </button>
            <button type="submit" className="save-button">
              {entry ? '更新' : '保存'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default PasswordForm;
