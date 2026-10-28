@echo off
chcp 65001 >nul
echo 启动后端服务器...
cd backend
go mod tidy
go run .
