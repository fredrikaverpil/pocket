@echo off
setlocal EnableDelayedExpansion

set "POK_DIR=.pocket"
set "POK_CONTEXT=."

rem Insert -v before -- if present, otherwise append it.
set "ARGS="
set "FOUND_SEP=0"
for %%a in (%*) do (
    if "%%a"=="--" (
        set "ARGS=!ARGS! -v --"
        set "FOUND_SEP=1"
    ) else (
        set "ARGS=!ARGS! %%a"
    )
)
if "!FOUND_SEP!"=="0" set "ARGS=!ARGS! -v"

go run -C "%POK_DIR%" .%ARGS%
