#include <windows.h>
static void NullifyParams(struct _DCB *params) {
/*	LPCTSTR str = "baud=115200 parity=N data=8 stop=1";

	BuildCommDCB(str,params);
*/
    params->fBinary       = TRUE;
    params->fNull         = FALSE;
    params->fErrorChar    = FALSE;
    params->fAbortOnError = FALSE;
}
