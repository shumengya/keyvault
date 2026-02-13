@echo off
chcp 65001 >nul
echo 构建前端项目...
cd mengyakeyvault-frontend
if not exist node_modules (
    echo 正在安装依赖...
    call npm install
)
npm run build
echo 构建完成！输出目录：mengyakeyvault-frontend\build
