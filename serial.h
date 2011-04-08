#include <windows.h>
static void NullifyParams(struct _DCB *params) {
    params->fBinary       = TRUE;
    params->fNull         = FALSE;
    params->fErrorChar    = FALSE;
    params->fAbortOnError = FALSE;
}
