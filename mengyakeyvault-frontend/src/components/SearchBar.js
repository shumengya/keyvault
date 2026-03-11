import React from 'react';
import './SearchBar.css';

// SVG 图标组件
const SearchIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="11" cy="11" r="8"></circle>
    <path d="m21 21-4.35-4.35"></path>
  </svg>
);

const ClearIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="18" y1="6" x2="6" y2="18"></line>
    <line x1="6" y1="6" x2="18" y2="18"></line>
  </svg>
);

const SearchBar = ({ keyword, onKeywordChange }) => {
  return (
    <div className="search-bar-container">
      <div className="search-bar">
        <span className="search-icon"><SearchIcon /></span>
        <input
          type="text"
          value={keyword}
          onChange={(e) => onKeywordChange(e.target.value)}
          placeholder="搜索官方名称、账号、用户名、邮箱、网站、标签..."
          className="search-input"
        />
        {keyword && (
          <button
            className="clear-button"
            onClick={() => onKeywordChange('')}
            title="清除搜索"
          >
            <ClearIcon />
          </button>
        )}
      </div>
    </div>
  );
};

export default SearchBar;
