package gothic

/*
#cgo LDFLAGS: -ltcl8.5 -ltk8.5
#cgo CFLAGS: -I/usr/include/tcl8.5

#include "interpreter.h"
*/
import "C"
import (
	"errors"
	"reflect"
	"runtime"
	"bytes"
	"unsafe"
	"image"
	"sync"
	"fmt"
)

const (
	debug = false
	alot  = 999999
)

//------------------------------------------------------------------------------
// Utils
//------------------------------------------------------------------------------

func cgo_string_to_go_string(p *C.char, n C.int) string {
	var x reflect.StringHeader
	x.Data = uintptr(unsafe.Pointer(p))
	x.Len = int(n)
	return *(*string)(unsafe.Pointer(&x))
}

func go_string_to_cgo_string(s string) (*C.char, C.int) {
	x := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	return (*C.char)(unsafe.Pointer(x.Data)), C.int(x.Len)
}

func c_interface_to_go_interface(iface [2]unsafe.Pointer) interface{} {
	return *(*interface{})(unsafe.Pointer(&iface))
}

func go_interface_to_c_interface(iface interface{}) *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(&iface))
}

// A handle that is used to manipulate a TCL interpreter. All handle methods
// can be safely invoked from different threads. Each method invokation is
// synchronous, it means that the method will be blocked until the action is
// actually executed.
type Interpreter struct {
	ir   *interpreter
	Done <-chan int
}

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

// Queue script for evaluation and wait for its completion. This function uses printf-like
// formatting style. However it provides a tiny wrapper on top of printf for the purpose of
// being friendly with TCL's syntax. Also it provides several advanced features like named
// and positional arguments.
//
// The syntax for formatting tags is %{<abbrev>[<format>]}, where:
//  - <abbrev> could be a number of the function argument (starting from 0) or
//    a name of the key in the provided gothic.ArgMap argument. It can also be
//    empty, in this case it uses internal counter, takes the corresponding
//    argument and increments that counter.
//  - <format> is the fmt.Sprintf format specifier, passed directly to
//    fmt.Sprintf as is.
//
// NOTES:
// 1. Formatter is extended to do TCL-specific quoting on %q format specifier.
// 2. Named abbrev is allowed when there is only one argument and the type of this
//    argument is gothic.ArgMap.
//
// Examples:
// 1. gothic.Sprintf("%{0} = %{1} + %{1}", 10, 5)
//    "10 = 5 + 5"
// 2. gothic.Sprintf("%{} = %{%d} + %{1}", 20, 10)
//    "20 = 10 + 10"
// 3. gothic.Sprintf("%{0%.2f} and %{%.2f}", 3.1415)
//    "3.14 and 3.14"
// 4. gothic.Sprintf("[myfunction %{arg1} %{arg2}]", gothic.ArgMap{
//            "arg1": 5,
//            "arg2": 10,
//    })
//    "[myfunction 5 10]"
// 5. gothic.Sprintf("%{%q}", "[command $variable]")
//    `"\[command \$variable\]"`
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

func (ir *Interpreter) Set(name string, val interface{}) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.set(name, val))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.set(name, val))
	})
}

func (ir *Interpreter) ErrorFilter(filt func(error)error) {
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

func (ir *Interpreter) RegisterCommand(name string, cbfunc interface{}) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.register_command(name, cbfunc))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.register_command(name, cbfunc))
	})
}

func (ir *Interpreter) UnregisterCommand(name string) error {
	if C.Tcl_GetCurrentThread() == ir.ir.thread {
		return ir.ir.filt(ir.ir.unregister_command(name))
	}
	return ir.ir.run_and_wait(func() error {
		return ir.ir.filt(ir.ir.unregister_command(name))
	})
}

//------------------------------------------------------------------------------
// interpreter
//------------------------------------------------------------------------------

type interpreter struct {
	C *C.Tcl_Interp

	errfilt func(error) error

	// registered commands
	commands map[string]interface{}

	// registered channels
	channels map[string]interface{}

	// just a buffer to avoid allocs in _gotk_go_command_handler
	valuesbuf []reflect.Value

	thread C.Tcl_ThreadId
	queue  chan async_action
	cmdbuf bytes.Buffer
}

func new_interpreter() (*interpreter, error) {
	ir := &interpreter{
		C:         C.Tcl_CreateInterp(),
		errfilt:   func(err error) error { return err },
		commands:  make(map[string]interface{}),
		channels:  make(map[string]interface{}),
		valuesbuf: make([]reflect.Value, 0, 10),
		queue:     make(chan async_action, 50),
		thread:    C.Tcl_GetCurrentThread(),
	}

	status := C.Tcl_Init(ir.C)
	if status != C.TCL_OK {
		return nil, errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}

	status = C.Tk_Init(ir.C)
	if status != C.TCL_OK {
		return nil, errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}

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
	obj = C.Tcl_SetVar2Ex(ir.C, cname, nil, obj, C.TCL_LEAVE_ERR_MSG)
	C.free(unsafe.Pointer(cname))
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
	C.free(unsafe.Pointer(cname))
	block := C.Tk_PhotoImageBlock{
		(*C.uchar)(unsafe.Pointer(&nrgba.Pix[0])),
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
func _gotk_go_command_handler(clidataup unsafe.Pointer, objc int, objv unsafe.Pointer) int {
	// TODO: There is an idea of optimizing everything by a large margin,
	// we can preprocess the type of a command in RegisterCommand function
	// and then avoid calling reflect.New for every argument passed to that
	// function. And we can even do additional error checks for unsupported
	// argument types and handle multiple return values case.

	clidata := (*C.GoTkClientData)(clidataup)
	ir := (*interpreter)(clidata.go_interp)
	args := (*(*[alot]*C.Tcl_Obj)(objv))[1:objc]
	cb := c_interface_to_go_interface(clidata.iface)
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

//export _gotk_go_command_deleter
func _gotk_go_command_deleter(data unsafe.Pointer) {
	clidata := (*C.GoTkClientData)(data)
	ir := (*interpreter)(clidata.go_interp)
	delete(ir.commands, cgo_string_to_go_string(clidata.strp, clidata.strn))
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
	cp, cn := go_string_to_cgo_string(name)
	cname := C.CString(name)
	C._gotk_c_add_command(ir.C, cname, unsafe.Pointer(ir), cp, cn,
		go_interface_to_c_interface(cbfunc))
	C.free(unsafe.Pointer(cname))
	return nil
}

func (ir *interpreter) unregister_command(name string) error {
	if _, ok := ir.commands[name]; !ok {
		return errors.New("gothic: trying to unregister a non-existent command")
	}
	cname := C.CString(name)
	status := C.Tcl_DeleteCommand(ir.C, cname)
	C.free(unsafe.Pointer(cname))
	if status != C.TCL_OK {
		return errors.New(C.GoString(C.Tcl_GetStringResult(ir.C)))
	}
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
	ev := C._gotk_c_new_async_event(unsafe.Pointer(ir))
	C.Tcl_ThreadQueueEvent(ir.thread, ev, C.TCL_QUEUE_TAIL)
	C.Tcl_ThreadAlert(ir.thread)

	// wait for result
	cond.Wait()
	cond.L.Unlock()

	return
}

//export _gotk_go_async_handler
func _gotk_go_async_handler(ev unsafe.Pointer, flags int) int {
	if flags != C.TK_ALL_EVENTS {
		return 0
	}
	event := (*C.GoTkAsyncEvent)(ev)
	ir := (*interpreter)(event.go_interp)
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
