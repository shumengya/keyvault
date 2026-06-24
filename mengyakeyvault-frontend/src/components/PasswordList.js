import React, { useState, useEffect } from 'react';
import './PasswordList.css';

// SVG 图标组件
const CopyIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
  </svg>
);

const CheckIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="20 6 9 17 4 12"></polyline>
  </svg>
);

const EditIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
  </svg>
);

const DeleteIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="3 6 5 6 21 6"></polyline>
    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
  </svg>
);

const LinkIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
  </svg>
);

const EmptyIcon = () => (
  <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
    <polyline points="14 2 14 8 20 8"></polyline>
    <line x1="12" y1="18" x2="12" y2="12"></line>
    <line x1="9" y1="15" x2="15" y2="15"></line>
  </svg>
);

// 与 PasswordList.css 网格列数保持一致，便于按「行数×列数」计算每页条数
const getGridColumnCount = (width) => {
  if (width <= 768) return 2;
  if (width <= 1199) return 3;
  if (width <= 1599) return 4;
  return 5;
};

const useGridColumnCount = () => {
  const [columns, setColumns] = useState(() => getGridColumnCount(window.innerWidth));
  useEffect(() => {
    const handler = () => setColumns(getGridColumnCount(window.innerWidth));
    window.addEventListener('resize', handler);
    return () => window.removeEventListener('resize', handler);
  }, []);
  return columns;
};

const PasswordList = ({ entries, onEdit, onDelete }) => {
  const [copiedId, setCopiedId] = useState(null);
  const [logoCache, setLogoCache] = useState({});
  const [currentPage, setCurrentPage] = useState(1);
  const gridColumns = useGridColumnCount();

  const ROWS_PER_PAGE = 4;
  const pageSize = ROWS_PER_PAGE * gridColumns;
  const totalPages = Math.ceil(entries.length / pageSize);

  // entries 变化时重置到第一页
  useEffect(() => {
    setCurrentPage(1);
  }, [entries]);

  // 列数变化（窗口缩放）时避免当前页超出总页数
  useEffect(() => {
    setCurrentPage((p) => {
      const tp = Math.max(1, Math.ceil(entries.length / pageSize) || 1);
      return Math.min(p, tp);
    });
  }, [pageSize, entries.length]);

  const pagedEntries = entries.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  // 获取网站favicon
  const getWebsiteFavicon = (url) => {
    if (!url) return null;
    try {
      const urlObj = new URL(url);
      const domain = urlObj.host;
      return `https://favicon.smyhub.com/api/favicon?url=${encodeURIComponent(domain)}`;
    } catch {
      return null;
    }
  };

  // 获取logo URL
  const getLogoUrl = (entry) => {
    // 如果entry中有logo字段且不为空，使用该logo
    if (entry.logo && entry.logo.trim() !== '') {
      return entry.logo;
    }
    
    // 如果有网站地址，尝试获取favicon
    if (entry.website) {
      const faviconUrl = getWebsiteFavicon(entry.website);
      if (faviconUrl) {
        return faviconUrl;
      }
    }

    // 默认使用本地logo
    return `${process.env.PUBLIC_URL}/logo.png`;
  };

  const handleCopy = async (text, id) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    } catch (err) {
      // 降级方案
      const textArea = document.createElement('textarea');
      textArea.value = text;
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand('copy');
      document.body.removeChild(textArea);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    }
  };

  const CopyButton = ({ text, id }) => {
    const uniqueId = `copy-${id}-${text}`;
    const isCopied = copiedId === uniqueId;
    return (
      <button
        className="copy-button"
        onClick={(e) => {
          e.stopPropagation();
          handleCopy(text, uniqueId);
        }}
        title={isCopied ? '已复制！' : '复制'}
      >
        {isCopied ? <CheckIcon /> : <CopyIcon />}
      </button>
    );
  };
  if (entries.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-icon"><EmptyIcon /></div>
        <p>暂无密码记录</p>
        <p className="empty-hint">点击"添加密码"按钮开始添加</p>
      </div>
    );
  }

  return (
    <>
    <div className="password-list">
      {pagedEntries.map((entry) => {
        const logoUrl = getLogoUrl(entry);
        return (
        <div key={entry.id} className="password-card">
          <div className="card-header">
            <div className="card-logo-wrapper">
              <img 
                src={logoUrl} 
                alt={entry.officialName || 'Logo'} 
                className="card-logo"
                onError={(e) => {
                  // 如果加载失败，使用默认logo
                  e.target.src = `${process.env.PUBLIC_URL}/logo.png`;
                }}
              />
            </div>
            <div className="card-title-section">
              <div className="card-title-row">
                <div className="card-title">
                  <span className="card-type">{entry.officialName || entry.software || '未命名'}</span>
                </div>
                <div className="card-actions">
                  <button
                    className="edit-button"
                    onClick={() => onEdit(entry)}
                    title="编辑"
                  >
                    <EditIcon />
                  </button>
                  <button
                    className="delete-button"
                    onClick={() => onDelete(entry.id)}
                    title="删除"
                  >
                    <DeleteIcon />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div className="card-content">
            {entry.account && (
              <div className="info-row">
                <span className="info-label">账号：</span>
                <span className="info-value">{entry.account}</span>
                <CopyButton text={entry.account} id={entry.id} />
              </div>
            )}
            {entry.password && (
              <div className="info-row">
                <span className="info-label">密码：</span>
                <span className="info-value password-value" title={entry.password}>
                  {entry.password}
                </span>
                <CopyButton text={entry.password} id={entry.id} />
              </div>
            )}
            {entry.username && (
              <div className="info-row">
                <span className="info-label">用户名：</span>
                <span className="info-value">{entry.username}</span>
                <CopyButton text={entry.username} id={entry.id} />
              </div>
            )}
            {entry.phone && (
              <div className="info-row">
                <span className="info-label">手机号：</span>
                <span className="info-value">{entry.phone}</span>
                <CopyButton text={entry.phone} id={entry.id} />
              </div>
            )}
            {entry.email && (
              <div className="info-row">
                <span className="info-label">邮箱：</span>
                <span className="info-value">{entry.email}</span>
                <CopyButton text={entry.email} id={entry.id} />
              </div>
            )}
            {entry.website && (
              <div className="info-row">
                <span className="info-label">网站：</span>
                <a
                  href={entry.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="info-link"
                  title={entry.website}
                >
                  {entry.website}
                </a>
                <CopyButton text={entry.website} id={entry.id} />
              </div>
            )}
          </div>
          {entry.tags && (
            <div className="card-tags">
              {entry.tags.split(',').map((tag, idx) => (
                <span key={idx} className="card-tag">{tag.trim()}</span>
              ))}
            </div>
          )}
        </div>
        );
      })}
    </div>
    {totalPages > 1 && (
      <div className="pagination">
        <button
          className="page-btn"
          onClick={() => setCurrentPage(1)}
          disabled={currentPage === 1}
          title="第一页"
        >«</button>
        <button
          className="page-btn"
          onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
          disabled={currentPage === 1}
          title="上一页"
        >‹</button>

        {Array.from({ length: totalPages }, (_, i) => i + 1)
          .filter(p => p === 1 || p === totalPages || Math.abs(p - currentPage) <= 2)
          .reduce((acc, p, idx, arr) => {
            if (idx > 0 && p - arr[idx - 1] > 1) acc.push('...');
            acc.push(p);
            return acc;
          }, [])
          .map((p, idx) =>
            p === '...'
              ? <span key={`ellipsis-${idx}`} className="page-ellipsis">…</span>
              : <button
                  key={p}
                  className={`page-btn${currentPage === p ? ' active' : ''}`}
                  onClick={() => setCurrentPage(p)}
                >{p}</button>
          )
        }

        <button
          className="page-btn"
          onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
          disabled={currentPage === totalPages}
          title="下一页"
        >›</button>
        <button
          className="page-btn"
          onClick={() => setCurrentPage(totalPages)}
          disabled={currentPage === totalPages}
          title="最后一页"
        >»</button>

        <span className="page-info">{currentPage} / {totalPages} 页 · 共 {entries.length} 条</span>
      </div>
    )}
    </>
  );
};

export default PasswordList;
