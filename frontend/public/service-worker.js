/* =====================================================
 * 萌芽密码管理器 Service Worker
 * 策略：
 *   - 静态资源（Shell）: Cache First（优先缓存）
 *   - API 请求:         Network First（优先网络，失败时返回离线页）
 *   - 导航请求:         Network First → 回退到缓存的 index.html
 * ===================================================== */

const CACHE_NAME = 'mengyakeyvault-v1';
const OFFLINE_URL = '/offline.html';

// 预缓存的应用 Shell 资源
const PRECACHE_URLS = [
  '/',
  '/index.html',
  '/offline.html',
  '/manifest.json',
  '/favicon.ico',
  '/logo.png',
];

// ── Install ──────────────────────────────────────────
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(PRECACHE_URLS).catch((err) => {
        console.warn('[SW] 预缓存部分资源失败:', err);
      });
    })
  );
  // 强制新 SW 立即激活，不等旧 SW 退出
  self.skipWaiting();
});

// ── Activate ─────────────────────────────────────────
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((name) => name !== CACHE_NAME)
          .map((name) => caches.delete(name))
      );
    })
  );
  // 立即接管所有页面
  self.clients.claim();
});

// ── Fetch ─────────────────────────────────────────────
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // 只处理 http/https，忽略 chrome-extension 等
  if (!url.protocol.startsWith('http')) return;

  // API 请求：Network First
  if (url.pathname.startsWith('/api') || url.hostname.includes('keyvault')) {
    event.respondWith(networkFirst(request));
    return;
  }

  // 第三方资源（favicon API 等）：Network First，不缓存
  if (url.hostname !== self.location.hostname) {
    event.respondWith(networkOnly(request));
    return;
  }

  // 导航请求（HTML页面）：Network First → 回退 index.html
  if (request.mode === 'navigate') {
    event.respondWith(navigationHandler(request));
    return;
  }

  // 静态资源（JS/CSS/图片等）：Cache First
  event.respondWith(cacheFirst(request));
});

// ── 策略函数 ──────────────────────────────────────────

// Cache First：先查缓存，没有再请求网络并写入缓存
async function cacheFirst(request) {
  const cached = await caches.match(request);
  if (cached) return cached;
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    return new Response('资源暂时无法访问', { status: 503 });
  }
}

// Network First：先请求网络，失败时查缓存
async function networkFirst(request) {
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await caches.match(request);
    return cached || new Response(
      JSON.stringify({ error: '网络不可用，请检查连接' }),
      { status: 503, headers: { 'Content-Type': 'application/json' } }
    );
  }
}

// Network Only：仅网络，不缓存
async function networkOnly(request) {
  try {
    return await fetch(request);
  } catch {
    return new Response('', { status: 503 });
  }
}

// 导航处理：Network First → 回退缓存的 index.html
async function navigationHandler(request) {
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await caches.match('/index.html');
    if (cached) return cached;
    return caches.match(OFFLINE_URL);
  }
}

// ── 消息处理（支持主线程主动触发更新）──────────────────
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
