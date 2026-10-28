// API 地址：通过 REACT_APP_API_BASE 在构建时注入
// - dist（网站版）: /api
// - dist-desktop（桌面版）: https://keyvault.shumengya.top/api
const API_BASE = process.env.REACT_APP_API_BASE || '/api';

export default API_BASE;