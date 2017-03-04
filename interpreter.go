package gothic

/*
#cgo !tcl85 LDFLAGS: -ltcl8.6 -ltk8.6
#cgo !tcl85 CFLAGS: -I/usr/include/tcl8.6
#cgo tcl85 LDFLAGS: -ltcl8.5 -ltk8.5
#cgo tcl85 CFLAGS: -I/usr/include/tcl8.5
#cgo darwin tcl85 CFLAGS: -I/opt/X11/include

#include "interpreter.h"
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

const (
	debug = false
	alot  = 999999
)

//------------------------------------------------------------------------------
// Utils
//------------------------------------------------------------------------------

// A handle that is used to manipulate a TCL interpreter. All handle methods
// can be safely invoked from different threads. Each method invocation is
// synchronous, it means that the method will be blocked until the action is
// actually executed.
//
// `Done` field returns 0 when Tk's main loop exits.
type Interpreter struct {
	ir   *interpreter
	Done <-chan int
}

// Creates a new instance of the *gothic.Interpreter. But before interpreter
// enters the Tk's main loop it will execute `init`. Init argument could be a
// string or a function with this signature: "func(*gothic.Interpreter)".
func NewInterpreter(init interface{}) *Interpreter {
	initdone := make(chan int)
	done := make(chan int)

	ir := new(Interpreter)
	ir.Done = done

	go func() {
		var err error
		runtime.LockOSThread()
		ir.ir, err = new_interpreter()
		if err != nil {
			panic(err)
		}

		switch realinit := init.(type) {
		case string:
			err = ir.ir.eval([]byte(realinit))
			if err != nil {
				panic(err)
			}
		case func(*Interpreter):
			realinit(ir)
		}

		initdone <- 0
		C.Tk_MainLoop()
		done <- 0
	}()

	<-initdone
	return ir
}

// Queue script for evaluation and wait for its completion. This function uses
// printf-like formatting style, except that all the formatting tags are
// enclosed within %{}. The reason for this is because tcl/tk uses %-based
// formatting tags for its own purposes. Also it provides several extensions
// like named and positional arguments. When no arguments are provided it will
// evaluate the format string as-is.
//
// Another important difference between fmt.Sprintf and Eval is that when using
// Eval, invalid format/arguments combination results in an error, while
// fmt.Sprintf simply ignores misconfiguration. All the formatter generated
// errors go through ErrorFilter, just like any other tcl error.
//
// The syntax for formatting tags is:
//  %{[<abbrev>[<format>]]}
//
// Where:
//
//  <abbrev> could be a number of the function argument (starting from 0) or a
//           name of the key in the provided gothic.ArgMap argument. It can
//           also be empty, in this case it uses internal counter, takes the
//           corresponding argument and increments that counter.
//
//  <format> Is the fmt.Sprintf format specifier, passed directly to
//           fmt.Sprintf as is (except for %q, see additional notes).
//
// Additional notes:
//
//  1. Formatter is extended to do TCL-specific quoting on %q format specifier.
//  2. Named abbrev is only allowed when there is one argument and the type of
//     this argument is gothic.ArgMap.
//
// Examples:
//  1. gothic.Eval("%{0} = %{1} + %{1}", 10, 5)
//     "10 = 5 + 5"
//  2. gothic.Eval("%{} = %{%d} + %{1}", 20, 10)
//     "20 = 10 + 10"
//  3. gothic.Eval("%{0%.2f} and %{%.2f}", 3.1415)
//     "3.14 and 3.14"
//  4. gothic.Eval("[myfunction %{arg1} %{arg2}]", gothic.ArgMap{
//             "arg1": 5,
//             "arg2": 10,
//     })
//     "[myfunction 5 10]"
//  5. gothic.Eval("%{%q}", "[command $variable]")
//     `"\[command \$variable\]"`
func (ir *Interpreter) Eval(format string, args ...interface{}) error {
	// interpreter thread
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		ir.ir.cmdbuf.Reset()
		err := sprintf(&ir.ir.cmdbuf, format, args...)
		if err != nil {
			return ir.ir.filt(err)
		}
		err = ir.ir.eval(ir.ir.cmdbuf.Bytes())
		return ir.ir.filt(err)
	}

	// foreign thread
	buf := buffer_pool.get()
	err := sprintf(&buf, format, args...)
	if err != nil {
		buffer_pool.put(buf)
		return ir.ir.filt(err)
	}
	script := buf.Bytes()
	err = ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.eval(script))
	})
	buffer_pool.put(buf)
	return err
}

// Works the same way as Eval("%{}", byte_slice), but avoids unnecessary
// buffering.
func (ir *Interpreter) EvalBytes(s []byte) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.eval(s))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.eval(s))
	})
}

// Works exactly as Eval with exception that it writes the result of executed
// code into `out`.
func (ir *Interpreter) EvalAs(out interface{}, format string, args ...interface{}) error {
	// interpreter thread
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		ir.ir.cmdbuf.Reset()
		err := sprintf(&ir.ir.cmdbuf, format, args...)
		if err != nil {
			return ir.ir.filt(err)
		}
		err = ir.ir.eval_as(out, ir.ir.cmdbuf.Bytes())
		return ir.ir.filt(err)
	}

	// foreign thread
	buf := buffer_pool.get()
	err := sprintf(&buf, format, args...)
	if err != nil {
		buffer_pool.put(buf)
		return ir.ir.filt(err)
	}
	script := buf.Bytes()
	err = ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.eval_as(out, script))
	})
	buffer_pool.put(buf)
	return err
}

// Shortcut for `var i int; err := EvalAs(&i, format, args...)`
func (ir *Interpreter) EvalAsInt(format string, args ...interface{}) (int, error) {
	var as int
	err := ir.EvalAs(&as, format, args...)
	return as, err
}

// Shortcut for `var s string; err := EvalAs(&s, format, args...)`
func (ir *Interpreter) EvalAsString(format string, args ...interface{}) (string, error) {
	var as string
	err := ir.EvalAs(&as, format, args...)
	return as, err
}

// Shortcut for `var f float64; err := EvalAs(&f, format, args...)`
func (ir *Interpreter) EvalAsFloat(format string, args ...interface{}) (float64, error) {
	var as float64
	err := ir.EvalAs(&as, format, args...)
	return as, err
}

// Shortcut for `var b bool; err := EvalAs(&b, format, args...)`
func (ir *Interpreter) EvalAsBool(format string, args ...interface{}) (bool, error) {
	var as bool
	err := ir.EvalAs(&as, format, args...)
	return as, err
}

// Sets the TCL variable `name` to the `val`. Sometimes it's nice to be able to
// avoid going through TCL's syntax. Especially for things like passing a whole
// buffer of text to TCL.
func (ir *Interpreter) Set(name string, val interface{}) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.set(name, val))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.set(name, val))
	})
}

// Every TCL error goes through the filter passed to this function. If you pass
// nil, then no error filter is set.
func (ir *Interpreter) ErrorFilter(filt func(error) error) {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		ir.ir.errfilt = filt
	}
	ir.ir.run_and_wait(func() error {
		ir.ir.errfilt = filt
		return nil
	})
}

func (ir *Interpreter) UploadImage(name string, img image.Image) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.upload_image(name, img))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.upload_image(name, img))
	})
}

// Register a new TCL command called `name`.
func (ir *Interpreter) RegisterCommand(name string, cbfunc interface{}) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.register_command(name, cbfunc))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.register_command(name, cbfunc))
	})
}

// Register multiple TCL command within the `name` namespace. The method uses
// runtime reflection and registers only those methods of the `val` which have
// one of the following prefixes: "TCL" or "TCL_". The name of the resulting
// command doesn't include the prefix.
func (ir *Interpreter) RegisterCommands(name string, val interface{}) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.register_commands(name, val))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.register_commands(name, val))
	})
}

// Unregisters (deletes) previously registered command `name`.
func (ir *Interpreter) UnregisterCommand(name string) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.unregister_command(name))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.unregister_command(name))
	})
}

// Unregisters (deletes) previously registered command set within the `name`
// namespace.
func (ir *Interpreter) UnregisterCommands(name string) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.unregister_commands(name))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.unregister_commands(name))
	})
}

//------------------------------------------------------------------------------
// interpreter
//------------------------------------------------------------------------------

type global_handles_t struct {
	sync.Mutex
	handles handles
}

func (g *global_handles_t) get_handle_for_value(value interface{}) int {
	g.Lock()
	out := g.handles.get_handle_for_value(value)
	g.Unlock()
	return out
}

func (g *global_handles_t) free_handle(id int) {
	g.Lock()
	g.free_handle(id)
	g.Unlock()
}

func (g *global_handles_t) get(id int) interface{} {
	g.Lock()
	out := g.handles[id].Value
	g.Unlock()
	return out
}

var global_handles global_handles_t

//------------------------------------------------------------------------------
// interpreter
//------------------------------------------------------------------------------

type interpreter struct {
	C *C.Tcl_Interp

	errfilt func(error) error

	// id of this interpreter in global_handles table
	id int

	// buffer for C -> Go handles, every Go object passed to C is stored here
	handles handles

	// registered commands
	commands map[string]interface{}

	// registered method sets
	methods        map[string]interface{}
	method_handles map[string][]int

	// just a buffer to avoid allocs in _gotk_go_command_handler
	valuesbuf []reflect.Value

	thread C.Tcl_ThreadId
	queue  chan async_action
	cmdbuf bytes.Buffer
}

func release_interpreter(ir *interpreter) {
	global_handles.free_handle(ir.id)
}

func new_interpreter() (*interpreter, error) {
	ir := &interpreter{
		C:              C.Tcl_CreateInterp(),
		errfilt:        func(err error) error { return err },
		commands:       make(map[string]interface{}),
		methods:        make(map[string]interface{}),
		method_handles: make(map[string][]int),
		valuesbuf:      make([]reflect.Value, 0, 10),
		queue:          make(chan async_action, 50),
		thread:         C.Tcl_GetCurrentThread(),
	}

	status := C.Tcl_Init(ir.C)
	if status != C.TCL_OK {
		return nil, errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}

	status = C.Tk_Init(ir.C)
	if status != C.TCL_OK {
		return nil, errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}

	ir.id = global_handles.get_handle_for_value(ir)
	runtime.SetFinalizer(ir, release_interpreter)
	return ir, nil
}

func (ir *interpreter) filt(err error) error {
	errfilt := ir.errfilt
	ir.errfilt = nil
	if errfilt != nil {
		err = errfilt(err)
	}
	ir.errfilt = errfilt
	return err
}

func (ir *interpreter) eval(script []byte) error {
	if len(script) == 0 {
		return nil
	}
	status := C.Tcl_EvalEx(ir.C, (*C.char)(unsafe.Pointer(&script[0])),
		C.int(len(script)), 0)
	if status != C.TCL_OK {
		return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	return nil
}

func (ir *interpreter) eval_as(out interface{}, script []byte) error {
	pv := reflect.ValueOf(out)
	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		panic("gothic: EvalAs expected a non-nil pointer argument")
	}
	v := pv.Elem()

	err := ir.eval(script)
	if err != nil {
		return err
	}

	return ir.tcl_obj_to_go_value(C.Tcl_GetObjResult(ir.C), v)
}

func go_value_to_tcl_obj(value interface{}) *C.Tcl_Obj {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return C.Tcl_NewWideIntObj(C.Tcl_WideInt(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return C.Tcl_NewWideIntObj(C.Tcl_WideInt(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return C.Tcl_NewDoubleObj(C.double(v.Float()))
	case reflect.Bool:
		if v.Bool() {
			return C.Tcl_NewBooleanObj(1)
		}
		return C.Tcl_NewBooleanObj(0)
	case reflect.String:
		s := v.String()
		sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
		return C.Tcl_NewStringObj((*C.char)(unsafe.Pointer(sh.Data)), C.int(len(s)))
	}
	return nil
}

func (ir *interpreter) set(name string, value interface{}) error {
	obj := go_value_to_tcl_obj(value)
	if obj == nil {
		return errors.New("gothic: cannot convert Go value to TCL object")
	}

	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	obj = C.Tcl_SetVar2Ex(ir.C, cname, nil, obj, C.TCL_LEAVE_ERR_MSG)
	if obj == nil {
		return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	return nil
}

func (ir *interpreter) upload_image(name string, img image.Image) error {
	var buf bytes.Buffer
	err := sprintf(&buf, "image create photo %{}", name)
	if err != nil {
		return err
	}

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
	defer C.free(unsafe.Pointer(cname))

	handle := C.Tk_FindPhoto(ir.C, cname)
	if handle == nil {
		err := ir.eval(buf.Bytes())
		if err != nil {
			return err
		}
		handle = C.Tk_FindPhoto(ir.C, cname)
		if handle == nil {
			return errors.New("failed to create an image handle")
		}
	}

	imgdata := C.CBytes(nrgba.Pix)
	defer C.free(imgdata)

	block := C.Tk_PhotoImageBlock{
		(*C.uchar)(imgdata),
		C.int(nrgba.Rect.Max.X),
		C.int(nrgba.Rect.Max.Y),
		C.int(nrgba.Stride),
		4,
		[...]C.int{0, 1, 2, 3},
	}

	status := C.Tk_PhotoPutBlock(ir.C, handle, &block, 0, 0,
		C.int(nrgba.Rect.Max.X), C.int(nrgba.Rect.Max.Y),
		C.TK_PHOTO_COMPOSITE_SET)
	if status != C.TCL_OK {
		return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	return nil
}

func (ir *interpreter) tcl_obj_to_go_value(obj *C.Tcl_Obj, v reflect.Value) error {
	var status C.int

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var out C.Tcl_WideInt
		status = C.Tcl_GetWideIntFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetInt(int64(out))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var out C.Tcl_WideInt
		status = C.Tcl_GetWideIntFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetUint(uint64(out))
		}
	case reflect.String:
		var n C.int
		out := C.Tcl_GetStringFromObj(obj, &n)
		v.SetString(C.GoStringN(out, n))
	case reflect.Float32, reflect.Float64:
		var out C.double
		status = C.Tcl_GetDoubleFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetFloat(float64(out))
		}
	case reflect.Bool:
		var out C.int
		status = C.Tcl_GetBooleanFromObj(ir.C, obj, &out)
		if status == C.TCL_OK {
			v.SetBool(out == 1)
		}
	default:
		return fmt.Errorf("gothic: cannot convert TCL object to Go type: %s", v.Type())
	}

	if status != C.TCL_OK {
		return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	return nil
}

//------------------------------------------------------------------------------
// interpreter.commands
//------------------------------------------------------------------------------

//export _gotk_go_command_handler
func _gotk_go_command_handler(clidataup unsafe.Pointer, objc C.int, objv unsafe.Pointer) C.int {
	// TODO: There is an idea of optimizing everything by a large margin,
	// we can preprocess the type of a command in RegisterCommand function
	// and then avoid calling reflect.New for every argument passed to that
	// function. And we can even do additional error checks for unsupported
	// argument types and handle multiple return values case.

	clidata := (*C.GoTkClientData)(clidataup)
	ir := global_handles.get(int(clidata.go_interp)).(*interpreter)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]
	cb := ir.handles[clidata.h0].Value
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

		v := reflect.New(in).Elem()
		err := ir.tcl_obj_to_go_value(args[i], v)
		if err != nil {
			C._gotk_c_tcl_set_result(ir.C, C.CString(err.Error()))
			return C.TCL_ERROR
		}

		ir.valuesbuf = append(ir.valuesbuf, v)
	}

	// TODO: handle return value
	f.Call(ir.valuesbuf)

	return C.TCL_OK
}

//export _gotk_go_method_handler
func _gotk_go_method_handler(clidataup unsafe.Pointer, objc C.int, objv unsafe.Pointer) C.int {
	// TODO: There is an idea of optimizing everything by a large margin,
	// we can preprocess the type of a command in RegisterCommand function
	// and then avoid calling reflect.New for every argument passed to that
	// function. And we can even do additional error checks for unsupported
	// argument types and handle multiple return values case.

	clidata := (*C.GoTkClientData)(clidataup)
	ir := global_handles.get(int(clidata.go_interp)).(*interpreter)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]
	recv := ir.handles[clidata.h0].Value
	meth := ir.handles[clidata.h1].Value
	f := reflect.ValueOf(meth)
	ft := f.Type()

	ir.valuesbuf = ir.valuesbuf[:0]
	ir.valuesbuf = append(ir.valuesbuf, reflect.ValueOf(recv))
	for i, n := 1, ft.NumIn(); i < n; i++ {
		ia := i - 1
		in := ft.In(i)

		// use default value, if there is not enough args
		if len(args) <= ia {
			ir.valuesbuf = append(ir.valuesbuf, reflect.New(in).Elem())
			continue
		}

		v := reflect.New(in).Elem()
		err := ir.tcl_obj_to_go_value(args[ia], v)
		if err != nil {
			C._gotk_c_tcl_set_result(ir.C, C.CString(err.Error()))
			return C.TCL_ERROR
		}

		ir.valuesbuf = append(ir.valuesbuf, v)
	}

	// TODO: handle return value
	f.Call(ir.valuesbuf)

	return C.TCL_OK
}

//export _gotk_go_command_deleter
func _gotk_go_command_deleter(data unsafe.Pointer) {
	clidata := (*C.GoTkClientData)(data)
	ir := global_handles.get(int(clidata.go_interp)).(*interpreter)
	ir.handles.free_handle(int(clidata.h0))
}

func (ir *interpreter) register_command(name string, cbfunc interface{}) error {
	typ := reflect.TypeOf(cbfunc)
	if typ.Kind() != reflect.Func {
		return errors.New("gothic: RegisterCommand only accepts func type as a second argument")
	}
	if _, ok := ir.commands[name]; ok {
		return errors.New("gothic: command with the same name was already registered")
	}
	ir.commands[name] = cbfunc
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C._gotk_c_add_command(ir.C, cname, C.int(ir.id), C.int(ir.handles.get_handle_for_value(cbfunc)))
	return nil
}

func (ir *interpreter) register_commands(name string, val interface{}) error {
	if _, ok := ir.methods[name]; ok {
		return errors.New("gothic: method set with the same name was already registered")
	}
	ir.methods[name] = val
	t := reflect.TypeOf(val)
	valh := ir.handles.get_handle_for_value(val)
	ir.method_handles[name] = append(ir.method_handles[name], valh)
	for i, n := 0, t.NumMethod(); i < n; i++ {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "TCL") {
			continue
		}

		subname := m.Name[3:]
		if strings.HasPrefix(m.Name, "TCL_") {
			subname = m.Name[4:]
		}

		cname := C.CString(name + "::" + subname)
		defer C.free(unsafe.Pointer(cname))

		mh := ir.handles.get_handle_for_value(m.Func.Interface())
		ir.method_handles[name] = append(ir.method_handles[name], mh)
		C._gotk_c_add_method(ir.C, cname, C.int(ir.id), C.int(valh), C.int(mh))
	}
	return nil
}

func (ir *interpreter) unregister_command(name string) error {
	if _, ok := ir.commands[name]; !ok {
		return errors.New("gothic: trying to unregister a non-existent command")
	}

	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	status := C.Tcl_DeleteCommand(ir.C, cname)
	if status != C.TCL_OK {
		return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
	delete(ir.commands, name)
	return nil
}

func (ir *interpreter) unregister_commands(name string) error {
	if _, ok := ir.methods[name]; !ok {
		return errors.New("gothic: trying to unregister a non-existent method set")
	}
	val := ir.methods[name]
	t := reflect.TypeOf(val)
	for i, n := 0, t.NumMethod(); i < n; i++ {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "TCL") {
			continue
		}

		subname := m.Name[3:]
		if strings.HasPrefix(m.Name, "TCL_") {
			subname = m.Name[4:]
		}

		cname := C.CString(name + "::" + subname)
		defer C.free(unsafe.Pointer(cname))

		status := C.Tcl_DeleteCommand(ir.C, cname)
		if status != C.TCL_OK {
			return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
		}
	}
	delete(ir.methods, name)
	for _, id := range ir.method_handles[name] {
		ir.handles.free_handle(id)
	}
	delete(ir.method_handles, name)
	return nil
}

//------------------------------------------------------------------------------
// interpreter.async
//------------------------------------------------------------------------------

type async_action struct {
	result *error
	action func() error
	cond   *sync.Cond
}

func (ir *interpreter) run_and_wait(action func() error) (err error) {
	cond := sync.NewCond(&sync.Mutex{})
	cond.L.Lock()

	// send event
	ir.queue <- async_action{result: &err, action: action, cond: cond}
	ev := C._gotk_c_new_async_event(C.int(ir.id))
	C.Tcl_ThreadQueueEvent(ir.thread, ev, C.TCL_QUEUE_TAIL)
	C.Tcl_ThreadAlert(ir.thread)

	// wait for result
	cond.Wait()
	cond.L.Unlock()

	return
}

//export _gotk_go_async_handler
func _gotk_go_async_handler(ev unsafe.Pointer, flags C.int) C.int {
	if flags != C.TK_ALL_EVENTS {
		return 0
	}
	event := (*C.GoTkAsyncEvent)(ev)
	ir := global_handles.get(int(event.go_interp)).(*interpreter)
	action := <-ir.queue
	if action.result == nil {
		action.action()
	} else {
		*action.result = action.action()
	}
	action.cond.L.Lock()
	action.cond.Signal()
	action.cond.L.Unlock()
	return 1
}
