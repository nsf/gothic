#include "interpreter.h"

static void free_string(char *c) {
	free(c);
}

void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result) {
	Tcl_SetResult(interp, result, free_string);
}

GoTkClientData *_gotk_c_client_data_new(int go_interp, int h0, int h1) {
	GoTkClientData *cd = malloc(sizeof(GoTkClientData));
	cd->go_interp = go_interp;
	cd->h0 = h0;
	cd->h1 = h1;
	return cd;
}

//------------------------------------------------------------------------------
// Command
//------------------------------------------------------------------------------

extern int _gotk_go_command_handler(GoTkClientData*, int, Tcl_Obj**);
extern int _gotk_go_method_handler(GoTkClientData*, int, Tcl_Obj**);
extern void _gotk_go_command_deleter(GoTkClientData*);

int _gotk_c_command_handler(ClientData cd, Tcl_Interp *interp, int objc, Tcl_Obj *CONST objv[]) {
	return _gotk_go_command_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

int _gotk_c_method_handler(ClientData cd, Tcl_Interp *interp, int objc, Tcl_Obj *CONST objv[]) {
	return _gotk_go_method_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

void _gotk_c_command_deleter(ClientData cd) {
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_command_deleter(clidata);
	free(cd);
}

void _gotk_c_method_deleter(ClientData cd) {
	free(cd);
}

void _gotk_c_add_command(Tcl_Interp *interp, const char *name, int go_interp, int f)
{
	GoTkClientData *cd = _gotk_c_client_data_new(go_interp, f, 0);
	Tcl_CreateObjCommand(interp, name, _gotk_c_command_handler,
			     (ClientData)cd, _gotk_c_command_deleter);
}

void _gotk_c_add_method(Tcl_Interp *interp, const char *name, int go_interp,
	int recv, int meth)
{
	GoTkClientData *cd = _gotk_c_client_data_new(go_interp, recv, meth);
	Tcl_CreateObjCommand(interp, name, _gotk_c_method_handler,
			     (ClientData)cd, _gotk_c_method_deleter);
}

//------------------------------------------------------------------------------
// Async
//------------------------------------------------------------------------------

extern int _gotk_go_async_handler(Tcl_Event*, int);

Tcl_Event *_gotk_c_new_async_event(int go_interp) {
	GoTkAsyncEvent *ev = (GoTkAsyncEvent*)Tcl_Alloc(sizeof(GoTkAsyncEvent));
	ev->header.proc = _gotk_go_async_handler;
	ev->header.nextPtr = 0;
	ev->go_interp = go_interp;
	return (Tcl_Event*)ev;
}
