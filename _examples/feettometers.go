package main

import "github.com/nsf/gothic"

const init_script = `
	wm title . "Feet to Meters"
	grid [ttk::frame .c -padding "3 3 12 12"] -column 0 -row 0 -sticky nwes
	grid columnconfigure . 0 -weight 1; grid rowconfigure . 0 -weight 1

	grid [ttk::entry .c.feet -width 7 -textvariable feet] -column 2 -row 1 -sticky we
	grid [ttk::label .c.meters -textvariable meters] -column 2 -row 2 -sticky we
	grid [ttk::button .c.calc -text "Calculate" -command calculate] -column 3 -row 3 -sticky w

	grid [ttk::label .c.flbl -text "feet"] -column 3 -row 1 -sticky w
	grid [ttk::label .c.islbl -text "is equivalent to"] -column 1 -row 2 -sticky e
	grid [ttk::label .c.mlbl -text "meters"] -column 3 -row 2 -sticky w

	foreach w [winfo children .c] {grid configure $w -padx 5 -pady 5}
	focus .c.feet
	bind . <Return> {calculate}
`

func main() {
	ir := gothic.NewInterpreter(init_script)
	ir.RegisterCommand("calculate", func() {
		var f float64
		ir.EvalAs(&f, "set feet")
		ir.Eval("set meters %{%.3f}", f * 0.3048)
	})

	<-ir.Done
}
