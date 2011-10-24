package main

import "github.com/nsf/gothic"
import "time"
import "fmt"

func main() {
	ir, err := gothic.NewInterpreter()
	if err != nil {
		panic(err)
	}

	go func(){
		i := 0
		inc := -1
		for {
			if i > 99 || i < 1 {
				inc = -inc
			}
			i += inc
			time.Sleep(5e7)
			ir.AsyncEval(fmt.Sprintf(`.bar configure -value %d`, i))
		}
	}()

	ir.Eval(`
pack [ttk::progressbar .bar] -padx 20 -pady 20
	`)
	ir.MainLoop()
}
