@echo off
setlocal
cd /d "%~dp0"

rem Keep this whole file ASCII-only. Under chcp 65001, non-ASCII text in this script
rem makes cmd misread the file after an external program returns and run text
rem fragments as commands. The exe prints its own (Chinese) messages.
rem
rem One-click start: build first if bin\mengyakeyvault.exe is missing, then run it.
rem Extra arguments are passed to the exe, e.g.  start.cmd -reset-passwords
rem Data dir defaults to backend\data (same as start_backend.cmd, so the old data.json
rem there gets imported). Set MENGYA_DATA_DIR beforehand to use another location.
rem Do not double-click bin\mengyakeyvault.exe directly: without MENGYA_DATA_DIR it
rem creates an empty database in bin\data.
if not defined MENGYA_DATA_DIR set "MENGYA_DATA_DIR=%~dp0backend\data"

if not exist "bin\mengyakeyvault.exe" (
    echo Not built yet, building first...
    rem call by full path: some shells set NoDefaultCurrentDirectoryInExePath=1,
    rem which stops cmd from finding build.cmd by bare name
    call "%~dp0build.cmd"
    if errorlevel 1 goto :fail
)

"bin\mengyakeyvault.exe" %*
if errorlevel 1 goto :fail

endlocal
exit /b 0

:fail
echo.
echo Failed, see the messages above.
rem pause so a double-clicked window stays open long enough to read the error
pause
endlocal
exit /b 1
