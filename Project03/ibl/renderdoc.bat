setlocal
set RENDERDOC_LIBRARY_PATH=d:/Programs/RenderDoc/renderdoc.dll
rem go test -c advanced-gl/Project03/ibl/ibl
ibl.test.exe -test.run "^TestConvert$"
endlocal
pause