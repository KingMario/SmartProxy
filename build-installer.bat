@echo off
setlocal

set APP_NAME=SmartProxy
set EXE_NAME=%APP_NAME%.exe

echo [1/3] Building Windows GUI executable...
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-H=windowsgui" -o %EXE_NAME% .
if errorlevel 1 (
  echo Build failed.
  exit /b 1
)

set ISCC_CMD=iscc
where /q %ISCC_CMD%
if errorlevel 1 (
  if exist "%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe" set ISCC_CMD="%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe"
)

echo [2/3] Building installer with Inno Setup...
%ISCC_CMD% installer\windows\SmartProxy.iss
if errorlevel 1 (
  echo Installer build failed. Ensure Inno Setup 6 is installed and ISCC is available.
  exit /b 1
)

echo [3/3] Done.
echo Output: dist\windows\SmartProxy-Installer.exe
exit /b 0
