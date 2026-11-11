@echo off
rem Builds Tinkerie.exe: all seven apps and the launcher in one file, no DLLs.
rem Needs Go 1.24+ from https://go.dev/dl/ (standard library only).
rem To redraw the icon first:  go run ./tools/mkicon  then
rem   go run ./tools/mksyso -ico build/tinkerie.ico -out rsrc_windows_amd64.syso -version 0.25.0-alpha
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-H windowsgui -s -w" -o Tinkerie.exe .
if errorlevel 1 (
  echo Build failed.
  exit /b 1
)
echo Built Tinkerie.exe
