// 项目已不再使用 PWA。保留这个文件只为一件事:
// 浏览器里残留的旧 Service Worker 会定期检查此地址,更新到这个版本后自行注销并清掉缓存,
// 否则旧的"缓存优先"策略会一直返回旧页面(包括已删除的"添加到桌面"横幅)。
// 等所有老用户都访问过一次之后,可以删除本文件。
self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const names = await caches.keys();
      await Promise.all(names.map((name) => caches.delete(name)));
      await self.registration.unregister();
      // 让已打开的页面重新加载,摆脱旧缓存
      const windows = await self.clients.matchAll({ type: 'window' });
      windows.forEach((client) => client.navigate(client.url));
    })()
  );
});
