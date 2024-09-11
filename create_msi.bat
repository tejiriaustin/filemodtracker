@echo off
REM create_msi.bat

REM Set paths
set WIX_PATH="C:\Program Files (x86)\WiX Toolset v3.11\bin"
set PROJECT_PATH=.

REM Build the Go binary
echo Building Go binary...
make run-service

REM Create MSI
echo Creating MSI...
%WIX_PATH%\candle.exe %PROJECT_PATH%\installer.wxs
%WIX_PATH%\light.exe -out FileModTracker.msi installer.wixobj

echo MSI creation complete.