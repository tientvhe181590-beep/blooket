# Interactive:  Windows  -> double-click build.bat  (shows status, pauses at end)
#                 Git Bash -> ./build.sh
# No pause:      build.bat nopause   or   ./build.sh nopause

.PHONY: build-win
build-win:
ifeq ($(OS),Windows_NT)
	cmd /c build.bat nopause
else
	chmod +x build.sh 2>/dev/null || true
	./build.sh nopause
endif
