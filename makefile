include $(GOROOT)/src/Make.inc

TARG=os/serial

ifeq ($(GOOS),windows)
CGOFILES=serial_windows.go
else
CGOFILES=serial_posix.go
endif

include $(GOROOT)/src/Make.pkg
