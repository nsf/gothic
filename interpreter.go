package gotk

/*
#cgo LDFLAGS: -ltcl8.5 -ltk8.5

#include <stdlib.h>
#include <tcl.h>
#include <tk.h>

typedef struct {
	void *go_interp;
	int slot;
} GoTkClientData;

typedef struct {
	GoTkClientData *clidata;
	int objc;
	Tcl_Obj **objv;
} GoTkCallbackData;

extern int _gotk_go_callback_handler(GoTkCallbackData*);
extern void _gotk_go_callback_deleter(GoTkClientData*);

static inline void free_string(char *s)
{
	free(s);
}

static int _gotk_c_callback_handler(ClientData cd, Tcl_Interp *interp,
				    int objc, Tcl_Obj *CONST objv[])
{
	GoTkCallbackData data = {(GoTkClientData*)cd, objc, (Tcl_Obj**)objv};
	return _gotk_go_callback_handler(&data);
}

static void _gotk_c_callback_deleter(ClientData cd)
{
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_callback_deleter(clidata);
	free(clidata);
}

static void _gotk_c_add_callback(Tcl_Interp *interp, const char *name,
				 void *go_interp, int slot)
{
	GoTkClientData *cd = malloc(sizeof(GoTkClientData));
	cd->go_interp = go_interp;
	cd->slot = slot;

	Tcl_CreateObjCommand(interp, name, _gotk_c_callback_handler,
			     (ClientData)cd, _gotk_c_callback_deleter);
}

static void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result)
{
	Tcl_SetResult(interp, result, free_string);
}
*/
import "C"
import (
	"reflect"
	"unsafe"
	"bytes"
	"os"
)

const (
	callbackPrefix = "GoTk::callback_"
	debug = true
)

//------------------------------------------------------------------------------
// StringVar
//------------------------------------------------------------------------------

type StringVar struct {
	data *C.char
	ir *Interpreter
	name string
}

func (sv *StringVar) Get() string {
	if sv.data == nil {
		return ""
	}
	return C.GoString(sv.data)
}

func (sv *StringVar) Set(s string) {
	if sv.data != nil {
		C.Tcl_Free(sv.data)
	}
	sv.data = C.Tcl_Alloc(C.uint(len(s) + 1))
	svslice := (*((*[999999]byte)(unsafe.Pointer(sv.data))))[:]
	copy(svslice, s)
	svslice[len(s)] = 0

	cname := C.CString(sv.name)
	C.Tcl_UpdateLinkedVar(sv.ir.C, cname)
	C.free_string(cname)
}

func (ir *Interpreter) NewStringVar(name string) *StringVar {
	sv := new(StringVar)
	sv.ir = ir
	sv.name = name
	sv.data = C.Tcl_Alloc(1)
	(*((*[999999]byte)(unsafe.Pointer(sv.data))))[0] = 0

	cname := C.CString(name)
	status := C.Tcl_LinkVar(ir.C, cname, (*C.char)(unsafe.Pointer(&sv.data)), C.TCL_LINK_STRING)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
	C.free_string(cname)
	return sv
}

//------------------------------------------------------------------------------
// FloatVar
//------------------------------------------------------------------------------

type FloatVar struct {
	data C.double
	ir *Interpreter
	name string
}

func (fv *FloatVar) Get() float64 {
	return float64(fv.data)
}

func (fv *FloatVar) Set(f float64) {
	fv.data = C.double(f)
	cname := C.CString(fv.name)
	C.Tcl_UpdateLinkedVar(fv.ir.C, cname)
	C.free_string(cname)
}

func (ir *Interpreter) NewFloatVar(name string) *FloatVar {
	fv := new(FloatVar)
	fv.ir = ir
	fv.name = name
	fv.data = 0.0

	cname := C.CString(name)
	status := C.Tcl_LinkVar(ir.C, cname, (*C.char)(unsafe.Pointer(&fv.data)), C.TCL_LINK_DOUBLE)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
	C.free_string(cname)
	return fv
}

//------------------------------------------------------------------------------
// Interpreter
//------------------------------------------------------------------------------

type Interpreter struct {
	C *C.Tcl_Interp

	// registered callbacks and the callback ID generator
	callbacks map[int]interface{}
	callbacksId int

	// just a buffer to avoid allocs in _gotk_go_callback_handler
	valuesbuf []reflect.Value

	// another buffer for Eval command construction
	cmdbuf bytes.Buffer
}

//export _gotk_go_callback_handler
func _gotk_go_callback_handler(data unsafe.Pointer) int {
	cbdata := (*C.GoTkCallbackData)(data)
	clidata := cbdata.clidata
	ir := (*Interpreter)(clidata.go_interp)
	args := (*(*[9999]*C.Tcl_Obj)(unsafe.Pointer(cbdata.objv)))[1:cbdata.objc]

	cb, ok := ir.callbacks[int(clidata.slot)]
	if !ok {
		msg := C.CString("Trying to invoke a non-existent callback")
		C._gotk_c_tcl_set_result(ir.C, msg)
		return C.TCL_ERROR
	}

	f := reflect.ValueOf(cb)
	if f.Kind() != reflect.Func {
		msg := C.CString("Trying to invoke a non-function callback")
		C._gotk_c_tcl_set_result(ir.C, msg)
		return C.TCL_ERROR
	}

	defer ir.resetValuesBuf()

	ft := f.Type()
	for i, n := 0, ft.NumIn(); i < n; i++ {
		in := ft.In(i)

		// use default value, if there is not enough args
		if len(args) <= i {
			ir.valuesbuf = append(ir.valuesbuf, reflect.New(in).Elem())
			continue
		}

		var status C.int
		var v reflect.Value

		switch in.Kind() {
		case reflect.Int:
			var out C.Tcl_WideInt
			status = C.Tcl_GetWideIntFromObj(ir.C, args[i], &out)
			if status != C.TCL_OK {
				return C.TCL_ERROR
			}

			v = reflect.New(in).Elem()
			v.SetInt(int64(out))
		case reflect.String:
			v = reflect.New(in).Elem()
			v.SetString(C.GoString(C.Tcl_GetString(args[i])))
		case reflect.Float32:
			var out C.double
			v = reflect.New(in).Elem()
			status = C.Tcl_GetDoubleFromObj(ir.C, args[i], &out)
			if status != C.TCL_OK {
				return C.TCL_ERROR
			}

			v.SetFloat(float64(out))
		case reflect.Float64:
			var out C.double
			v = reflect.New(in).Elem()
			status = C.Tcl_GetDoubleFromObj(ir.C, args[i], &out)
			if status != C.TCL_OK {
				return C.TCL_ERROR
			}

			v.SetFloat(float64(out))
		default:
			msg := C.CString("Fail")
			C._gotk_c_tcl_set_result(ir.C, msg)
			return C.TCL_ERROR
		}

		ir.valuesbuf = append(ir.valuesbuf, v)
	}

	f.Call(ir.valuesbuf)

	return C.TCL_OK
}

//export _gotk_go_callback_deleter
func _gotk_go_callback_deleter(data unsafe.Pointer) {
	clidata := (*C.GoTkClientData)(data)
	ir := (*Interpreter)(clidata.go_interp)
	ir.callbacks[int(clidata.slot)] = nil, false
}

func NewInterpreter() (*Interpreter, os.Error) {
	ir := &Interpreter{
		C: C.Tcl_CreateInterp(),
		callbacks: make(map[int]interface{}),
		valuesbuf: make([]reflect.Value, 0, 10),
	}

	status := C.Tcl_Init(ir.C)
	if status != C.TCL_OK {
		return nil, os.NewError(C.GoString(ir.C.result))
	}

	status = C.Tk_Init(ir.C)
	if status != C.TCL_OK {
		return nil, os.NewError(C.GoString(ir.C.result))
	}

	err := ir.Eval("namespace eval GoTk {}")
	if err != nil {
		return nil, err
	}
	return ir, nil
}

func (ir *Interpreter) resetValuesBuf() {
	ir.valuesbuf = ir.valuesbuf[:0]
}

func (ir *Interpreter) nextCallbackId() int {
	ret := ir.callbacksId
	ir.callbacksId++
	return ret
}

func (ir *Interpreter) Eval(args ...string) os.Error {
	for _, arg := range args {
		ir.cmdbuf.WriteString(arg)
		ir.cmdbuf.WriteString(" ")
	}

	s := ir.cmdbuf.String()
	ir.cmdbuf.Reset()

	if debug {
		println(s)
	}

	cs := C.CString(s)
	status := C.Tcl_Eval(ir.C, cs)
	C.free_string(cs)
	if status != C.TCL_OK {
		return os.NewError(C.GoString(ir.C.result))
	}
	return nil
}

func (ir *Interpreter) RegisterCallback(name string, cbfunc interface{}) {
	id := ir.nextCallbackId()
	ir.callbacks[id] = cbfunc

	cname := C.CString(name)
	C._gotk_c_add_callback(ir.C, cname, unsafe.Pointer(ir), C.int(id))
	C.free_string(cname)
}

func (ir *Interpreter) MainLoop() {
	C.Tk_MainLoop()
}
