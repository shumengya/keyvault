@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"

rem 用法: build.bat          编译 Windows 可执行文件 -> bin\mengyakeyvault.exe
rem       build.bat linux    交叉编译 Linux amd64 -> bin\mengyakeyvault-linux-amd64

set "NAME=mengyakeyvault.exe"
if /i "%~1"=="linux" (
    set "GOOS=linux"
    set "GOARCH=amd64"
    set "NAME=mengyakeyvault-linux-amd64"
)

echo [1/2] 构建前端，输出到 backend\web\dist，稍后被内嵌进二进制...
cd frontend
if not exist node_modules (
    echo 正在安装前端依赖...
    call npm install
    if errorlevel 1 goto :fail
)
call npm run build
if errorlevel 1 goto :fail
cd ..

echo [2/2] 编译后端...
if not exist bin mkdir bin
rem 使用纯 Go 的 SQLite 驱动，不需要 CGO，所以能直接交叉编译
set "CGO_ENABLED=0"
cd backend
go build -trimpath -ldflags "-s -w" -o "..\bin\%NAME%" .
if errorlevel 1 goto :fail
cd ..

echo.
echo 构建完成: bin\%NAME%
echo 直接运行它，然后用浏览器打开 http://localhost:8080 即可。
endlocal
exit /b 0

:fail
echo.
echo 构建失败，见上方报错。
endlocal
exit /b 1
