@echo off
taskkill /f /im course-helper.exe 2>nul
if %errorlevel% equ 0 (
    echo Course Helper has been stopped.
) else (
    echo Course Helper is not running.
)
pause
