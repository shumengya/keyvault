// 前端由后端二进制内嵌提供,接口与页面同源。
// 登录态靠 SameSite=Strict 的 Cookie,必须同源,所以这里不再支持配置别的后端地址。
const API_BASE = '/api';

export default API_BASE;
