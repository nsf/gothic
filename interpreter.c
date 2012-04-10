#include "interpreter.h"

static void free_string(char *c) {
	free(c);
}

void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result) {
	Tcl_SetResult(interp, result, free_string);
}

GoTkClientData *_gotk_c_client_data_new(void *go_interp, char *strp, int strn, void **iface) {
	GoTkClientData *cd = malloc(sizeof(GoTkClientData));
	cd->go_interp = go_interp;
	cd->strp = strp;
	cd->strn = strn;
	cd->iface[0] = iface[0];
	cd->iface[1] = iface[1];
	return cd;
}

//------------------------------------------------------------------------------
// Command
//------------------------------------------------------------------------------

extern int _gotk_go_command_handler(GoTkClientData*, int, Tcl_Obj**);
extern void _gotk_go_command_deleter(GoTkClientData*);


int _gotk_c_command_handler(ClientData cd, Tcl_Interp *interp, int objc, Tcl_Obj *CONST objv[]) {
	return _gotk_go_command_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

void _gotk_c_command_deleter(ClientData cd) {
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_command_deleter(clidata);
	free(cd);
}

void _gotk_c_add_command(Tcl_Interp *interp, const char *name, void *go_interp,
	char *strp, int strn, void **iface)
{
	GoTkClientData *cd = _gotk_c_client_data_new(go_interp, strp, strn, iface);
	Tcl_CreateObjCommand(interp, name, _gotk_c_command_handler,
			     (ClientData)cd, _gotk_c_command_deleter);
}

//------------------------------------------------------------------------------
// Channel
//------------------------------------------------------------------------------

extern int _gotk_go_channel_handler(GoTkClientData*, int, Tcl_Obj**);
extern void _gotk_go_channel_deleter(GoTkClientData*);

int _gotk_c_channel_handler(ClientData cd, Tcl_Interp *interp, int objc, Tcl_Obj *CONST objv[]) {
	return _gotk_go_channel_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

void _gotk_c_channel_deleter(ClientData cd) {
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_channel_deleter(clidata);
	free(cd);
}

void _gotk_c_add_channel(Tcl_Interp *interp, const char *name, void *go_interp,
	char *strp, int strn, void **iface)
{
	GoTkClientData *cd = _gotk_c_client_data_new(go_interp, strp, strn, iface);
	Tcl_CreateObjCommand(interp, name, _gotk_c_channel_handler,
			     (ClientData)cd, _gotk_c_channel_deleter);
}

//------------------------------------------------------------------------------
// Async
//------------------------------------------------------------------------------

extern int _gotk_go_async_handler(Tcl_Event*, int);

Tcl_Event *_gotk_c_new_async_event(void *go_interp) {
	GoTkAsyncEvent *ev = (GoTkAsyncEvent*)Tcl_Alloc(sizeof(GoTkAsyncEvent));
	ev->header.proc = _gotk_go_async_handler;
	ev->header.nextPtr = 0;
	ev->go_interp = go_interp;
	return (Tcl_Event*)ev;
}
