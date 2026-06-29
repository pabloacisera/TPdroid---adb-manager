@echo off
cd /d "%~dp0"
start "" /min "tpdroid.exe"
timeout /t 2 /nobreak >nul
start "" "http://localhost:8080"
