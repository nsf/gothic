package gotk

/*
#cgo LDFLAGS: -ltcl8.5 -ltk8.5

#include <stdlib.h>
#include <tcl.h>
#include <tk.h>

typedef struct {
	void *go_interp;
	char *p; // go string ptr
	int n;   // go string len
} GoTkClientData;

static inline void free_string(char *s)
{
	free(s);
}

static inline void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result)
{
	Tcl_SetResult(interp, result, free_string);
}

//------------------------------------------------------------------------------
// Callbacks
//------------------------------------------------------------------------------

extern int _gotk_go_callback_handler(GoTkClientData*, int, Tcl_Obj**);
extern void _gotk_go_callback_deleter(GoTkClientData*);

static int _gotk_c_callback_handler(ClientData cd, Tcl_Interp *interp,
				    int objc, Tcl_Obj *CONST objv[])
{
	return _gotk_go_callback_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

static void _gotk_c_callback_deleter(ClientData cd)
{
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_callback_deleter(clidata);
	free(cd);
}

static void _gotk_c_add_callback(Tcl_Interp *interp, const char *name,
				 void *go_interp, char *p, int n)
{
	GoTkClientData *cd = malloc(sizeof(GoTkClientData));
	cd->go_interp = go_interp;
	cd->p = p;
	cd->n = n;

	Tcl_CreateObjCommand(interp, name, _gotk_c_callback_handler,
			     (ClientData)cd, _gotk_c_callback_deleter);
}

//------------------------------------------------------------------------------
// Channels
//------------------------------------------------------------------------------

extern int _gotk_go_channel_handler(GoTkClientData*, int, Tcl_Obj**);
extern void _gotk_go_channel_deleter(GoTkClientData*);

static int _gotk_c_channel_handler(ClientData cd, Tcl_Interp *interp,
				   int objc, Tcl_Obj *CONST objv[])
{
	return _gotk_go_channel_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

static void _gotk_c_channel_deleter(ClientData cd)
{
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_channel_deleter(clidata);
	free(cd);
}

static void _gotk_c_add_channel(Tcl_Interp *interp, const char *name,
				void *go_interp, char *p, int n)
{
	GoTkClientData *cd = malloc(sizeof(GoTkClientData));
	cd->go_interp = go_interp;
	cd->p = p;
	cd->n = n;

	Tcl_CreateObjCommand(interp, name, _gotk_c_channel_handler,
			     (ClientData)cd, _gotk_c_channel_deleter);
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
	debug = true
	alot  = 999999
)

//------------------------------------------------------------------------------
// Utils
//------------------------------------------------------------------------------

func _CGoStringToGoString(p *C.char, n C.int) string {
	var x reflect.StringHeader
	x.Data = uintptr(unsafe.Pointer(p))
	x.Len = int(n)
	return *(*string)(unsafe.Pointer(&x))
}

func _GoStringToCGoString(s string) (*C.char, C.int) {
	x := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	return (*C.char)(unsafe.Pointer(x.Data)), C.int(x.Len)
}

//------------------------------------------------------------------------------
// StringVar
//------------------------------------------------------------------------------

type StringVar struct {
	data *C.char
	ir   *Interpreter
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
	svslice := (*((*[alot]byte)(unsafe.Pointer(sv.data))))[:]
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
	(*((*[alot]byte)(unsafe.Pointer(sv.data))))[0] = 0

	cname := C.CString(name)
	status := C.Tcl_LinkVar(ir.C, cname, (*C.char)(unsafe.Pointer(&sv.data)),
		C.TCL_LINK_STRING)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
	return sv
}

//------------------------------------------------------------------------------
// FloatVar
//------------------------------------------------------------------------------

type FloatVar struct {
	data C.double
	ir   *Interpreter
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
	status := C.Tcl_LinkVar(ir.C, cname, (*C.char)(unsafe.Pointer(&fv.data)),
		C.TCL_LINK_DOUBLE)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
	return fv
}

//------------------------------------------------------------------------------
// IntVar
//------------------------------------------------------------------------------

type IntVar struct {
	data C.Tcl_WideInt
	ir   *Interpreter
	name string
}

func (iv *IntVar) Get() int {
	return int(iv.data)
}

func (iv *IntVar) Set(i int) {
	iv.data = C.Tcl_WideInt(i)
	cname := C.CString(iv.name)
	C.Tcl_UpdateLinkedVar(iv.ir.C, cname)
	C.free_string(cname)
}

func (ir *Interpreter) NewIntVar(name string) *IntVar {
	iv := new(IntVar)
	iv.ir = ir
	iv.name = name
	iv.data = 0

	cname := C.CString(name)
	status := C.Tcl_LinkVar(ir.C, cname, (*C.char)(unsafe.Pointer(&iv.data)),
		C.TCL_LINK_WIDE_INT)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
	return iv
}

//------------------------------------------------------------------------------
// Interpreter
//------------------------------------------------------------------------------

type Interpreter struct {
	C *C.Tcl_Interp

	// registered callbacks
	callbacks map[string]interface{}

	// registered channels
	channels map[string]interface{}

	// just a buffer to avoid allocs in _gotk_go_callback_handler
	valuesbuf []reflect.Value

	// another buffer for Eval command construction
	cmdbuf bytes.Buffer
}

func NewInterpreter() (*Interpreter, os.Error) {
	ir := &Interpreter{
		C:         C.Tcl_CreateInterp(),
		callbacks: make(map[string]interface{}),
		channels:  make(map[string]interface{}),
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

	return ir, nil
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

func (ir *Interpreter) MainLoop() {
	C.Tk_MainLoop()
}

func (ir *Interpreter) tclObjToGoValue(obj *C.Tcl_Obj, typ reflect.Type) reflect.Value {
	v := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.Int:
		var out C.Tcl_WideInt
		status := C.Tcl_GetWideIntFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetInt(int64(out))
		}
	case reflect.String:
		v.SetString(C.GoString(C.Tcl_GetString(obj)))
	case reflect.Float32, reflect.Float64:
		var out C.double
		status := C.Tcl_GetDoubleFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetFloat(float64(out))
		}
	}
	return v
}

//------------------------------------------------------------------------------
// Interpreter.Callbacks
//------------------------------------------------------------------------------

//export _gotk_go_callback_handler
func _gotk_go_callback_handler(clidataup unsafe.Pointer, objc int, objv unsafe.Pointer) int {
	clidata := (*C.GoTkClientData)(clidataup)
	ir := (*Interpreter)(clidata.go_interp)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]

	cb, ok := ir.callbacks[_CGoStringToGoString(clidata.p, clidata.n)]
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

	ir.valuesbuf = ir.valuesbuf[:0]

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
	ir.callbacks[_CGoStringToGoString(clidata.p, clidata.n)] = nil, false
}

func (ir *Interpreter) RegisterCallback(name string, cbfunc interface{}) {
	ir.callbacks[name] = cbfunc
	cp, cn := _GoStringToCGoString(name)
	cname := C.CString(name)
	C._gotk_c_add_callback(ir.C, cname, unsafe.Pointer(ir), cp, cn)
	C.free_string(cname)
}

func (ir *Interpreter) UnregisterCallback(name string) {
	if _, ok := ir.callbacks[name]; !ok {
		return
	}
	cname := C.CString(name)
	status := C.Tcl_DeleteCommand(ir.C, cname)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
}

//------------------------------------------------------------------------------
// Interpreter.Channels
//------------------------------------------------------------------------------

//export _gotk_go_channel_handler
func _gotk_go_channel_handler(clidataup unsafe.Pointer, objc int, objv unsafe.Pointer) int {
	clidata := (*C.GoTkClientData)(clidataup)
	ir := (*Interpreter)(clidata.go_interp)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]
	if len(args) != 2 {
		msg := C.CString("gosend: argument count mismatch, expected two: <- VALUE")
		C._gotk_c_tcl_set_result(ir.C, msg)
		return C.TCL_ERROR
	}

	name := _CGoStringToGoString(clidata.p, clidata.n)

	var ch interface{}
	var ok bool
	if ch, ok = ir.channels[name]; !ok {
		msg := C.CString("gosend: trying to send to a non-existent channel")
		C._gotk_c_tcl_set_result(ir.C, msg)
		return C.TCL_ERROR
	}

	val := reflect.ValueOf(ch)
	arg := ir.tclObjToGoValue(args[1], val.Type().Elem())
	val.Send(arg)
	return C.TCL_OK
}

//export _gotk_go_channel_deleter
func _gotk_go_channel_deleter(data unsafe.Pointer) {
	clidata := (*C.GoTkClientData)(data)
	ir := (*Interpreter)(clidata.go_interp)
	ir.channels[_CGoStringToGoString(clidata.p, clidata.n)] = nil, false
}

func (ir *Interpreter) RegisterChannel(name string, ch interface{}) {
	ir.channels[name] = ch
	cp, cn := _GoStringToCGoString(name)
	cname := C.CString(name)
	C._gotk_c_add_channel(ir.C, cname, unsafe.Pointer(ir), cp, cn)
	C.free_string(cname)
}

func (ir *Interpreter) UnregisterChannel(name string) {
	if _, ok := ir.channels[name]; !ok {
		return
	}
	cname := C.CString(name)
	status := C.Tcl_DeleteCommand(ir.C, cname)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(ir.C.result))
	}
}
