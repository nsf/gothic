package gothic

/*
#cgo LDFLAGS: -ltcl8.5 -ltk8.5
#cgo CFLAGS: -I/usr/include/tcl8.5

#include <stdlib.h>
#include <tcl.h>
#include <tk.h>

typedef struct {
	void *go_interp; // go tcl/tk interpreter
	char *strp;      // go string ptr
	int strn;        // go string len
	void *iface[2];  // go interface
} GoTkClientData;

static inline void free_string(char *s)
{
	free(s);
}

static inline void _gotk_c_tcl_set_result(Tcl_Interp *interp, char *result)
{
	Tcl_SetResult(interp, result, free_string);
}

static inline GoTkClientData *_gotk_c_client_data_new(
	void *go_interp,
	char *strp,
	int strn,
	void **iface)
{
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

static int _gotk_c_command_handler(ClientData cd, Tcl_Interp *interp,
				    int objc, Tcl_Obj *CONST objv[])
{
	return _gotk_go_command_handler((GoTkClientData*)cd, objc, (Tcl_Obj**)objv);
}

static void _gotk_c_command_deleter(ClientData cd)
{
	GoTkClientData *clidata = (GoTkClientData*)cd;
	_gotk_go_command_deleter(clidata);
	free(cd);
}

static void _gotk_c_add_command(Tcl_Interp *interp, const char *name,
				void *go_interp, char *strp, int strn,
				void **iface)
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
				void *go_interp, char *strp, int strn,
				void **iface)
{
	GoTkClientData *cd = _gotk_c_client_data_new(go_interp, strp, strn, iface);
	Tcl_CreateObjCommand(interp, name, _gotk_c_channel_handler,
			     (ClientData)cd, _gotk_c_channel_deleter);
}

//------------------------------------------------------------------------------
// Async
//------------------------------------------------------------------------------

typedef struct {
	Tcl_Event header;
	void *go_interp;
} GoTkAsyncEvent;

extern int _gotk_go_async_handler(Tcl_Event*, int);

static Tcl_Event *_gotk_c_new_async_event(void *go_interp)
{
	GoTkAsyncEvent *ev = (GoTkAsyncEvent*)Tcl_Alloc(sizeof(GoTkAsyncEvent));
	ev->header.proc = _gotk_go_async_handler;
	ev->header.nextPtr = 0;
	ev->go_interp = go_interp;
	return (Tcl_Event*)ev;
}
*/
import "C"
import (
	"reflect"
	"runtime"
	"unsafe"
	"bytes"
	"image"
	"sync"
	"fmt"
	"os"
)

const (
	debug = false
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

func _CInterfaceToGoInterface(iface [2]unsafe.Pointer) interface{} {
	return *(*interface{})(unsafe.Pointer(&iface))
}

func _GoInterfacetoCInterface(iface interface{}) *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(&iface))
}

//------------------------------------------------------------------------------
// Interpreter
//
// A handle that is used to manipulate interpreter. All handle methods can be
// safely invoked from different threads. Each method invokation is
// synchornous, it means that the method will be blocked until the action is
// executed.
//------------------------------------------------------------------------------

type Interpreter struct {
	ir *interpreter
	Done <-chan int
}

func NewInterpreter(init string) *Interpreter {
	var ir *interpreter

	initdone := make(chan int)
	done := make(chan int)

	go func() {
		var err os.Error
		runtime.LockOSThread()
		ir, err = newInterpreter()
		if err != nil {
			panic(err)
		}

		ir.eval(init)
		initdone <- 0
		ir.mainLoop()
		done <- 0
	}()

	<-initdone
	return &Interpreter{ir, done}
}

func (ir *Interpreter) Eval(args ...interface{}) {
	ir.run(func() { ir.ir.eval(args...) })
}

func (ir *Interpreter) EvalAsString(args ...interface{}) (out string) {
	ir.runAndWait(func() { out = ir.ir.evalAsString(args...) })
	return
}

func (ir *Interpreter) EvalAsInt(args ...interface{}) (out int) {
	ir.runAndWait(func() { out = ir.ir.evalAsInt(args...) })
	return
}

func (ir *Interpreter) EvalAsFloat(args ...interface{}) (out float64) {
	ir.runAndWait(func() { out = ir.ir.evalAsFloat(args...) })
	return
}

func (ir *Interpreter) UploadImage(name string, img image.Image) {
	ir.run(func() { ir.ir.uploadImage(name, img) })
}

func (ir *Interpreter) RegisterCommand(name string, cbfunc interface{}) {
	ir.run(func() { ir.ir.registerCommand(name, cbfunc) })
}

func (ir *Interpreter) UnregisterCommand(name string) {
	ir.run(func() { ir.ir.unregisterCommand(name) })
}

func (ir *Interpreter) RegisterChannel(name string, ch interface{}) {
	ir.run(func() { ir.ir.registerChannel(name, ch) })
}

func (ir *Interpreter) UnregisterChannel(name string) {
	ir.run(func() { ir.ir.unregisterChannel(name) })
}

func (ir *Interpreter) Sync() {
	runtime.LockOSThread()
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return
	}
	runtime.UnlockOSThread()

	var m sync.Mutex
	c := sync.NewCond(&m)
	m.Lock()
	ir.ir.async(nil, c)
	c.Wait()
	m.Unlock()
}

func (ir *Interpreter) run(clo func()) {
	runtime.LockOSThread()
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		clo()
		return
	}
	runtime.UnlockOSThread()
	ir.ir.async(clo, nil)
}

func (ir *Interpreter) runAndWait(clo func()) {
	runtime.LockOSThread()
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		clo()
		return
	}
	runtime.UnlockOSThread()

	var m sync.Mutex
	c := sync.NewCond(&m)
	m.Lock()
	ir.ir.async(clo, c)
	c.Wait()
	m.Unlock()
}

//------------------------------------------------------------------------------
// interpreter
//------------------------------------------------------------------------------

type interpreter struct {
	C *C.Tcl_Interp

	// registered commands
	commands map[string]interface{}

	// registered channels
	channels map[string]interface{}

	// just a buffer to avoid allocs in _gotk_go_command_handler
	valuesbuf []reflect.Value

	thread C.Tcl_ThreadId
	queue chan asyncAction

	cmdbuf bytes.Buffer
}

func newInterpreter() (*interpreter, os.Error) {
	ir := &interpreter{
		C: C.Tcl_CreateInterp(),
		commands: make(map[string]interface{}),
		channels: make(map[string]interface{}),
		valuesbuf: make([]reflect.Value, 0, 10),
		queue: make(chan asyncAction, 50),
		thread: C.Tcl_GetCurrentThread(),
	}

	status := C.Tcl_Init(ir.C)
	if status != C.TCL_OK {
		return nil, os.NewError(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}

	status = C.Tk_Init(ir.C)
	if status != C.TCL_OK {
		return nil, os.NewError(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}

	return ir, nil
}

func (ir *interpreter) eval(args ...interface{}) {
	ir.cmdbuf.Reset()
	fmt.Fprint(&ir.cmdbuf, args...)

	if debug {
		println(ir.cmdbuf.String())
	}

	ir.cmdbuf.WriteByte(0)

	status := C.Tcl_Eval(ir.C, (*C.char)(unsafe.Pointer(&ir.cmdbuf.Bytes()[0])))
	if status != C.TCL_OK {
		fmt.Fprintln(os.Stderr, C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
}

func (ir *interpreter) evalAsString(args ...interface{}) string {
	ir.eval(args...)

	var n C.int
	str := C.Tcl_GetStringFromObj(C.Tcl_GetObjResult(ir.C), &n)
	return C.GoStringN(str, n)
}

func (ir *interpreter) evalAsInt(args ...interface{}) int {
	ir.eval(args...)

	var i C.Tcl_WideInt
	status := C.Tcl_GetWideIntFromObj(ir.C, C.Tcl_GetObjResult(ir.C), &i)
	if status != C.TCL_OK {
		fmt.Fprintln(os.Stderr, C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	return int(i)
}

func (ir *interpreter) evalAsFloat(args ...interface{}) float64 {
	ir.eval(args...)

	var f C.double
	status := C.Tcl_GetDoubleFromObj(ir.C, C.Tcl_GetObjResult(ir.C), &f)
	if status != C.TCL_OK {
		fmt.Fprintln(os.Stderr, C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	return float64(f)
}

func (ir *interpreter) mainLoop() {
	C.Tk_MainLoop()
}

func (ir *interpreter) uploadImage(name string, img image.Image) {
	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		// let's do it slowpoke
		bounds := img.Bounds()
		nrgba = image.NewNRGBA(bounds)
		for x := 0; x < bounds.Max.X; x++ {
			for y := 0; y < bounds.Max.Y; y++ {
				nrgba.Set(x, y, img.At(x, y))
			}
		}
	}

	cname := C.CString(name)
	handle := C.Tk_FindPhoto(ir.C, cname)
	if handle == nil {
		ir.eval("image create photo ", name)
		handle = C.Tk_FindPhoto(ir.C, cname)
		if handle == nil {
			panic("something terrible has happened")
		}
	}
	C.free_string(cname)
	block := C.Tk_PhotoImageBlock{
		(*C.uchar)(unsafe.Pointer(&nrgba.Pix[0])),
		C.int(nrgba.Rect.Max.X),
		C.int(nrgba.Rect.Max.Y),
		C.int(nrgba.Stride),
		4,
		[...]C.int{0, 1, 2, 3},
	}

	C.Tk_PhotoPutBlock_NoComposite(handle, &block, 0, 0,
		C.int(nrgba.Rect.Max.X), C.int(nrgba.Rect.Max.Y))

}

func (ir *interpreter) tclObjToGoValue(obj *C.Tcl_Obj, typ reflect.Type) (reflect.Value, C.int) {
	var status C.int
	v := reflect.New(typ).Elem()

	switch typ.Kind() {
	case reflect.Int:
		var out C.Tcl_WideInt
		status = C.Tcl_GetWideIntFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetInt(int64(out))
		}
	case reflect.String:
		v.SetString(C.GoString(C.Tcl_GetString(obj)))
	case reflect.Float32, reflect.Float64:
		var out C.double
		status = C.Tcl_GetDoubleFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetFloat(float64(out))
		}
	default:
		msg := C.CString(fmt.Sprintf("Cannot convert Tcl object to Go type: %s", typ))
		C._gotk_c_tcl_set_result(ir.C, msg)
		status = C.TCL_ERROR
	}
	return v, status
}

//------------------------------------------------------------------------------
// interpreter.commands
//------------------------------------------------------------------------------

//export _gotk_go_command_handler
func _gotk_go_command_handler(clidataup unsafe.Pointer, objc int, objv unsafe.Pointer) int {
	clidata := (*C.GoTkClientData)(clidataup)
	ir := (*interpreter)(clidata.go_interp)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]
	cb := _CInterfaceToGoInterface(clidata.iface)
	f := reflect.ValueOf(cb)
	ft := f.Type()

	ir.valuesbuf = ir.valuesbuf[:0]
	for i, n := 0, ft.NumIn(); i < n; i++ {
		in := ft.In(i)

		// use default value, if there is not enough args
		if len(args) <= i {
			ir.valuesbuf = append(ir.valuesbuf, reflect.New(in).Elem())
			continue
		}

		v, status := ir.tclObjToGoValue(args[i], in)
		if status != C.TCL_OK {
			return C.TCL_ERROR
		}

		ir.valuesbuf = append(ir.valuesbuf, v)
	}

	f.Call(ir.valuesbuf)

	return C.TCL_OK
}

//export _gotk_go_command_deleter
func _gotk_go_command_deleter(data unsafe.Pointer) {
	clidata := (*C.GoTkClientData)(data)
	ir := (*interpreter)(clidata.go_interp)
	ir.commands[_CGoStringToGoString(clidata.strp, clidata.strn)] = nil, false
}

func (ir *interpreter) registerCommand(name string, cbfunc interface{}) {
	typ := reflect.TypeOf(cbfunc)
	if typ.Kind() != reflect.Func {
		panic("RegisterCommand only accepts functions as a second argument")
	}
	ir.commands[name] = cbfunc
	cp, cn := _GoStringToCGoString(name)
	cname := C.CString(name)
	C._gotk_c_add_command(ir.C, cname, unsafe.Pointer(ir), cp, cn,
		_GoInterfacetoCInterface(cbfunc))
	C.free_string(cname)
}

func (ir *interpreter) unregisterCommand(name string) {
	if _, ok := ir.commands[name]; !ok {
		return
	}
	cname := C.CString(name)
	status := C.Tcl_DeleteCommand(ir.C, cname)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
}

//------------------------------------------------------------------------------
// interpreter.channels
//------------------------------------------------------------------------------

//export _gotk_go_channel_handler
func _gotk_go_channel_handler(clidataup unsafe.Pointer, objc int, objv unsafe.Pointer) int {
	clidata := (*C.GoTkClientData)(clidataup)
	ir := (*interpreter)(clidata.go_interp)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]
	if len(args) != 2 {
		msg := C.CString("Argument count mismatch, expected two: <- VALUE")
		C._gotk_c_tcl_set_result(ir.C, msg)
		return C.TCL_ERROR
	}

	ch := _CInterfaceToGoInterface(clidata.iface)
	val := reflect.ValueOf(ch)
	arg, status := ir.tclObjToGoValue(args[1], val.Type().Elem())
	if status != C.TCL_OK {
		return C.TCL_ERROR
	}

	val.Send(arg)
	return C.TCL_OK
}

//export _gotk_go_channel_deleter
func _gotk_go_channel_deleter(data unsafe.Pointer) {
	clidata := (*C.GoTkClientData)(data)
	ir := (*interpreter)(clidata.go_interp)
	ir.channels[_CGoStringToGoString(clidata.strp, clidata.strn)] = nil, false
}

func (ir *interpreter) registerChannel(name string, ch interface{}) {
	typ := reflect.TypeOf(ch)
	if typ.Kind() != reflect.Chan {
		panic("RegisterChannel only accepts channels as a second argument")
	}

	ir.channels[name] = ch
	cp, cn := _GoStringToCGoString(name)
	cname := C.CString(name)
	C._gotk_c_add_channel(ir.C, cname, unsafe.Pointer(ir), cp, cn,
		_GoInterfacetoCInterface(ch))
	C.free_string(cname)
}

func (ir *interpreter) unregisterChannel(name string) {
	if _, ok := ir.channels[name]; !ok {
		return
	}
	cname := C.CString(name)
	status := C.Tcl_DeleteCommand(ir.C, cname)
	C.free_string(cname)
	if status != C.TCL_OK {
		panic(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
}

//------------------------------------------------------------------------------
// interpreter.async
//------------------------------------------------------------------------------

type asyncAction struct {
	action func()
	cond *sync.Cond
}

func (ir *interpreter) async(action func(), cond *sync.Cond) {
	ir.queue <- asyncAction{action, cond}
	ev := C._gotk_c_new_async_event(unsafe.Pointer(ir))
	C.Tcl_ThreadQueueEvent(ir.thread, ev, C.TCL_QUEUE_TAIL)
	C.Tcl_ThreadAlert(ir.thread)
}

//export _gotk_go_async_handler
func _gotk_go_async_handler(ev unsafe.Pointer, flags int) int {
	event := (*C.GoTkAsyncEvent)(ev)
	ir := (*interpreter)(event.go_interp)
	action := <-ir.queue
	if action.action != nil {
		action.action()
	}
	if action.cond != nil {
		action.cond.L.Lock()
		action.cond.Signal()
		action.cond.L.Unlock()
	}
	return 1
}
