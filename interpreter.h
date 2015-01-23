#include <stdlib.h>
#include <tcl.h>
#include <tk.h>

typedef struct {
	int go_interp; // go tcl/tk interpreter handle, that's a global handle
	int h0;        // first handle to Go object (callback or receiver)
	int h1;        // second handle to Go object (method if h0 is receiver)
} GoTkClientData;


void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result);
GoTkClientData *_gotk_c_client_data_new(int go_interp, int h0, int h1);

//------------------------------------------------------------------------------
// Command
//------------------------------------------------------------------------------

int _gotk_c_command_handler(ClientData cd, Tcl_Interp *interp, int objc, Tcl_Obj *CONST objv[]);
void _gotk_c_command_deleter(ClientData cd);
void _gotk_c_add_command(Tcl_Interp *interp, const char *name, int go_interp, int cb);
void _gotk_c_add_method(Tcl_Interp *interp, const char *name, int go_interp,
	int recv, int meth);

//------------------------------------------------------------------------------
// Async
//------------------------------------------------------------------------------

typedef struct {
	Tcl_Event header;
	int go_interp;
} GoTkAsyncEvent;

Tcl_Event *_gotk_c_new_async_event(int go_interp);
