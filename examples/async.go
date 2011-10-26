package main

import "github.com/nsf/gothic"
import "time"

func main() {
	ir, err := gothic.NewInterpreter()
	if err != nil {
		panic(err)
	}

	go func() {
		// here I use AsyncEval
		i := 0
		inc := -1
		for {
			if i > 99 || i < 1 {
				inc = -inc
			}
			i += inc
			time.Sleep(5e7)
			ir.AsyncEval(`.bar1 configure -value`, i)
		}
	}()

	go func() {
		// here the Async generic action is used
		i := 0
		inc := -1
		closure := func() {
			ir.Eval(`.bar2 configure -value`, i)
		}

		for {
			if i > 99 || i < 1 {
				inc = -inc
			}
			i += inc
			time.Sleep(1e8)
			ir.Async(closure, nil, nil)
		}
	}()

	ir.Eval(`
pack [ttk::progressbar .bar1] -padx 20 -pady 20
pack [ttk::progressbar .bar2] -padx 20 -pady 20
	`)
	ir.MainLoop()
}
