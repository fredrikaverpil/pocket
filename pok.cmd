@echo off
setlocal EnableDelayedExpansion

set "POK_DIR=../.pocket"
set "POK_CONTEXT=v2"

go run -C "%POK_DIR%" . %*
