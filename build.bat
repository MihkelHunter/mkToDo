@echo off
REM build.bat — packages the ToDo app using fyne package
REM Run from the project root: build.bat
REM Requires: fyne CLI → go install fyne.io/fyne/v2/cmd/fyne@latest

set ICON=Icon.png
set APP_ID=com.mihkelhunter.todo
set APP_NAME=ToDo
set CMD_DIR=cmd\desktop

REM ── Windows ─────────────────────────────────────────────────────────────────
echo Building Windows...
cd %CMD_DIR%
set GOOS=windows
set GOARCH=amd64
fyne package -os windows -icon ..\..\%ICON% -app-id %APP_ID% -name %APP_NAME% -release true
cd ..\..
move /Y %CMD_DIR%\%APP_NAME%.exe .\Build\ToDo.exe
echo   ^-^> ToDo.exe

REM ── Linux (uncomment to enable) ──────────────────────────────────────────────
REM echo Building Linux...
REM cd %CMD_DIR%
REM set GOOS=linux
REM set GOARCH=amd64
REM fyne package -os linux -icon ..\..\%ICON% -app-id %APP_ID% -name %APP_NAME% -relese true
REM cd ..\..
REM move /Y %CMD_DIR%\%APP_NAME% .\Build\ToDo
REM echo   ^-^> ToDo

echo Done!
