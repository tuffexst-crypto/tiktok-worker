@echo off
title EXST ONLINE - TikTok Views Automation Worker
color 0b
echo ===================================================================
echo               EXST ONLINE - AUTOMATED STOREFRONT WORKER
echo                   Auto-Dispatching TikTok Views
echo ===================================================================
echo.
echo [*] Connecting to https://exststocks.org API...
echo [*] Worker is running in background and listening for paid orders.
echo [*] When a customer buys TikTok views, this window will automatically:
echo       1. Resolve the TikTok video ID
echo       2. Dispatch views via multi-threaded TikTok engine
echo       3. Complete and mark the order as fulfilled on the website
echo.
echo [*] Press Ctrl+C at any time to pause or stop.
echo.

go run main.go -worker -site https://exststocks.org -key EXST_Crypto_HMAC_Secret_991283_Uncrackable

pause
