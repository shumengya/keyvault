@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"

rem Usage: build.cmd          build Windows exe -> bin\mengyakeyvault.exe
rem        build.cmd linux    cross-compile Linux amd64 -> bin\mengyakeyvault-linux-amd64

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
rem pure-Go SQLite driver, no CGO needed, so cross-compiling just works
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
