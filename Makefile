include $(GOROOT)/src/Make.inc

TARG=os/serial

ifeq ($(GOOS),windows)
GOFILES=serial_$(GOOS).go
else
CGOFILES=serial_$(GOOS).go
endif

include $(GOROOT)/src/Make.pkg
