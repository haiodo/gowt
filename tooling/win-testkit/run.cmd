@echo off
setlocal enableextensions
rem gowt Windows test kit. run.cmd = SWT test gate; run.cmd dpi = multi-monitor DPI check.
cd /d "%~dp0"

set ARCH=%PROCESSOR_ARCHITECTURE%
if defined PROCESSOR_ARCHITEW6432 set ARCH=%PROCESSOR_ARCHITEW6432%
if /i "%ARCH%"=="ARM64" (set DIR=arm64) else if /i "%ARCH%"=="AMD64" (set DIR=amd64) else (
  echo Unsupported architecture: %ARCH%
  exit /b 2
)

set MODE=swttest
if /i "%~1"=="dpi" set MODE=dpi
set OUT=%~dp0result-%MODE%-%COMPUTERNAME%-%DIR%.txt

echo gowt Windows test kit, mode %MODE% > "%OUT%"
echo date: %DATE% %TIME% >> "%OUT%"
echo arch: %ARCH% (using %DIR%) >> "%OUT%"
ver >> "%OUT%"
powershell -NoProfile -Command "$v=Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion'; 'windows: ' + $v.ProductName + ' ' + $v.DisplayVersion + ' build ' + $v.CurrentBuild + '.' + $v.UBR" >> "%OUT%" 2>&1
powershell -NoProfile -Command "Add-Type -AssemblyName System.Windows.Forms; foreach($s in [System.Windows.Forms.Screen]::AllScreens){'screen: ' + $s.DeviceName + ' ' + $s.Bounds.Width + 'x' + $s.Bounds.Height + ' primary=' + $s.Primary}" >> "%OUT%" 2>&1

rem WebView2 runtime: the keys webview/wv2_util.go reads (ClientState EBWebView per user, then Clients pv).
set WV2=Microsoft\EdgeUpdate\ClientState\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}
set WV2C=Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}
echo webview2 registry: >> "%OUT%"
reg query "HKCU\SOFTWARE\%WV2%" /v EBWebView /reg:32 >> "%OUT%" 2>&1
reg query "HKLM\SOFTWARE\%WV2%" /v EBWebView /reg:32 >> "%OUT%" 2>&1
reg query "HKCU\SOFTWARE\%WV2C%" /v pv /reg:32 >> "%OUT%" 2>&1
reg query "HKLM\SOFTWARE\%WV2C%" /v pv /reg:32 >> "%OUT%" 2>&1

echo winprobe -display (DPI, awareness, monitors): >> "%OUT%"
"%~dp0%DIR%\winprobe.exe" -display >> "%OUT%" 2>&1

if "%MODE%"=="dpi" goto dpi

echo.
echo Running the SWT tests (%DIR%). Test windows will open and close by themselves. Ctrl+C stops.
echo ---- swttest output ---- >> "%OUT%"
"%~dp0%DIR%\swttest.exe" -expected "%~dp0expected_windows.txt" >> "%OUT%" 2>&1
set RC=%ERRORLEVEL%
echo ---- swttest exit code %RC% (0 = gate ok) ---- >> "%OUT%"
echo swttest exit code %RC%
goto done

:dpi
echo.
echo DPI check: the gallery window opens. Follow README.txt, section DPI check, then close the window.
"%~dp0%DIR%\controlexample.exe"
echo ---- after the window was closed ---- >> "%OUT%"
"%~dp0%DIR%\winprobe.exe" -display >> "%OUT%" 2>&1
echo.
echo Scales of the monitors (Settings - System - Display - Scale), for example "100%% and 150%%":
set /p SCALES=Scales: 
echo user scales: %SCALES% >> "%OUT%"
echo Did controls, fonts and window size rescale after dragging to the other monitor (y/n/describe)?
set /p DRAG=Drag: 
echo user drag: %DRAG% >> "%OUT%"
echo Did they rescale after changing the scale in Settings while it ran (y/n/describe)?
set /p CHG=Settings: 
echo user settings change: %CHG% >> "%OUT%"

:done
echo.
echo Result file, send it back:
echo %OUT%
pause
endlocal
