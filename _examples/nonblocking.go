package main

import "github.com/nsf/gothic"
import "time"

func proc(ir *gothic.Interpreter, num string) {
	button := ".b" + num
	progress := ".p" + num
	channame := "proc" + num
	recv := make(chan int)

	// register channel and enable button
	ir.RegisterChannel(channame, recv)
	ir.Eval(button, ` configure -state normal`)

	for {
		// wait for an event
		<-recv

		// simulate activity
		ir.Eval(button, ` configure -state disabled -text "In Progress `, num, `"`)
		for i := 0; i <= 100; i += 2 {
			ir.Eval(progress, ` configure -value `, i)
			time.Sleep(5e7)
		}

		// reset button state and progress value
		ir.Eval(progress, ` configure -value 0`)
		ir.Eval(button, ` configure -state normal -text "Start `, num, `"`)
	}
}

func main() {
	ir := gothic.NewInterpreter(`
foreach n {1 2 3} row {0 1 2} {
	ttk::button .b$n -text "Start $n" -command "proc$n <- 0" -state disabled -width 10
	grid .b$n -column 0 -row $row -padx 2 -pady 2 -sticky nwse
	grid [ttk::progressbar .p$n] -column 1 -row $row -padx 2 -pady 2 -sticky nwse
	grid rowconfigure . $row -weight 1
}
foreach col {0 1} { grid columnconfigure . $col -weight 1 }
	`)
	go proc(ir, "1")
	go proc(ir, "2")
	go proc(ir, "3")
	<-ir.Done
}
