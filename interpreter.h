#include <stdlib.h>
#include <tcl.h>
#include <tk.h>

typedef struct {
	void *go_interp; // go tcl/tk interpreter
	char *strp;      // go string ptr
	int strn;        // go string len
	void *iface[2];  // go interface
	void *iface2[2]; // go interface (used for receiver/method pair)
} GoTkClientData;


void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result);
GoTkClientData *_gotk_c_client_data_new(void *go_interp, char *strp, int strn, void **iface, void **iface2);

//------------------------------------------------------------------------------
// Command
//------------------------------------------------------------------------------

int _gotk_c_command_handler(ClientData cd, Tcl_Interp *interp, int objc, Tcl_Obj *CONST objv[]);
void _gotk_c_command_deleter(ClientData cd);
void _gotk_c_add_command(Tcl_Interp *interp, const char *name, void *go_interp,
	char *strp, int strn, void **iface);
void _gotk_c_add_method(Tcl_Interp *interp, const char *name, void *go_interp,
	void **iface, void **iface2);

//------------------------------------------------------------------------------
// Async
//------------------------------------------------------------------------------

typedef struct {
	Tcl_Event header;
	void *go_interp;
} GoTkAsyncEvent;

Tcl_Event *_gotk_c_new_async_event(void *go_interp);
