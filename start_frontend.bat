@echo off
chcp 65001 >nul
echo 启动前端开发服务器...
cd mengyakeyvault-frontend
if not exist node_modules (
    echo 正在安装依赖...
    call npm install
)
npm start
