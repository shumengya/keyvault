import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './PasswordManager.css';
import PasswordList from './PasswordList';
import PasswordForm from './PasswordForm';
import SearchBar from './SearchBar';
import SettingsPanel from './SettingsPanel';
import API_BASE from '../config/api';

const PasswordManager = ({ onLogout }) => {
  const [entries, setEntries] = useState([]);
  const [filteredEntries, setFilteredEntries] = useState([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [editingEntry, setEditingEntry] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadEntries();
  }, []);

  useEffect(() => {
    filterEntries();
  }, [searchKeyword, entries]);

  const loadEntries = async () => {
    try {
      setLoading(true);
      const response = await axios.get(`${API_BASE}/entries`);
      setEntries(response.data.entries || []);
    } catch (error) {
      console.error('加载条目失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const filterEntries = () => {
    if (!searchKeyword.trim()) {
      setFilteredEntries(entries);
      return;
    }

    const keyword = searchKeyword.toLowerCase();
    const filtered = entries.filter(entry =>
      entry.account?.toLowerCase().includes(keyword) ||
      entry.username?.toLowerCase().includes(keyword) ||
      entry.email?.toLowerCase().includes(keyword) ||
      entry.website?.toLowerCase().includes(keyword) ||
      entry.officialName?.toLowerCase().includes(keyword) ||
      (entry.software && entry.software.toLowerCase().includes(keyword)) ||
      entry.tags?.toLowerCase().includes(keyword)
    );
    setFilteredEntries(filtered);
  };

  const handleAdd = () => {
    setEditingEntry(null);
    setShowForm(true);
  };

  const handleEdit = (entry) => {
    setEditingEntry(entry);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定要删除这条记录吗？')) {
      return;
    }

    try {
      await axios.delete(`${API_BASE}/entries/${id}`);
      loadEntries();
    } catch (error) {
      console.error('删除失败:', error);
      alert('删除失败，请重试');
    }
  };

  const handleSave = async (entryData) => {
    try {
      if (editingEntry) {
        await axios.put(`${API_BASE}/entries`, { ...entryData, id: editingEntry.id });
      } else {
        await axios.post(`${API_BASE}/entries`, entryData);
      }
      setShowForm(false);
      setEditingEntry(null);
      loadEntries();
    } catch (error) {
      console.error('保存失败:', error);
      alert('保存失败，请重试');
    }
  };

  const handleCancel = () => {
    setShowForm(false);
    setEditingEntry(null);
  };

  if (loading) {
    return (
      <div className="manager-loading">
        <div className="loading-spinner"></div>
      </div>
    );
  }

  return (
    <div className="password-manager">
      <nav className="manager-nav">
        <div className="nav-content">
          <img 
            src={`${process.env.PUBLIC_URL}/logo.png`} 
            alt="Logo" 
            className="nav-logo"
          />
          <h1 className="nav-title">萌芽密码管理器</h1>
          <button
            type="button"
            className="nav-settings-button"
            onClick={() => setShowSettings(true)}
          >
            ⚙ 设置
          </button>
        </div>
      </nav>
      <div className="manager-header">
        <button className="add-button" onClick={handleAdd}>
          + 添加密码
        </button>
      </div>

      <SearchBar
        keyword={searchKeyword}
        onKeywordChange={setSearchKeyword}
      />

      <PasswordList
        entries={filteredEntries}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />

      {showForm && (
        <PasswordForm
          entry={editingEntry}
          onSave={handleSave}
          onCancel={handleCancel}
        />
      )}

      {showSettings && (
        <SettingsPanel onClose={() => setShowSettings(false)} onLogout={onLogout} />
      )}
    </div>
  );
};

export default PasswordManager;
