package main

import "github.com/nsf/gothic"
import "math/big"

var args [2]*big.Int
var lastOp string
var afterOp = true

func applyOp(op string, ir *gothic.Interpreter) {
	var num string
	ir.EvalAs(&num, "set calcText")
	if args[0] == nil {
		if op != "=" {
			args[0] = big.NewInt(0)
			args[0].SetString(num, 10)
		}
	} else {
		args[1] = big.NewInt(0)
		args[1].SetString(num, 10)
	}

	afterOp = true

	if args[1] == nil {
		lastOp = op
		return
	}

	switch lastOp {
	case "+":
		args[0] = args[0].Add(args[0], args[1])
	case "-":
		args[0] = args[0].Sub(args[0], args[1])
	case "/":
		args[0] = args[0].Div(args[0], args[1])
	case "*":
		args[0] = args[0].Mul(args[0], args[1])
	}

	lastOp = op
	args[1] = nil

	ir.Eval("set calcText %{}", args[0])
	if op == "=" {
		args[0] = nil
	}
}

func main() {
	ir := gothic.NewInterpreter(`
set lastOp {}
set calcText 0
wm title . "GoCalculator"

ttk::frame .f
ttk::entry .f.lastop -textvariable lastOp -justify center -state readonly -width 3
ttk::entry .f.entry -textvariable calcText -justify right -state readonly

grid .f.lastop .f.entry -sticky we
grid columnconfigure .f 0 -weight 0
grid columnconfigure .f 1 -weight 1

ttk::button .0 -text 0 -command { appendNum 0 }
ttk::button .1 -text 1 -command { appendNum 1 }
ttk::button .2 -text 2 -command { appendNum 2 }
ttk::button .3 -text 3 -command { appendNum 3 }
ttk::button .4 -text 4 -command { appendNum 4 }
ttk::button .5 -text 5 -command { appendNum 5 }
ttk::button .6 -text 6 -command { appendNum 6 }
ttk::button .7 -text 7 -command { appendNum 7 }
ttk::button .8 -text 8 -command { appendNum 8 }
ttk::button .9 -text 9 -command { appendNum 9 }
ttk::button .pm    -text +/- -command plusMinus
ttk::button .clear -text C -command clearAll
ttk::button .eq    -text = -command { applyOp = }
ttk::button .plus  -text + -command { applyOp + }
ttk::button .minus -text - -command { applyOp - }
ttk::button .mul   -text * -command { applyOp * }
ttk::button .div   -text / -command { applyOp / }

grid .f -   -      .div   -sticky nwes
grid .7 .8  .9     .mul   -sticky nwes
grid .4 .5  .6     .minus -sticky nwes
grid .1 .2  .3     .plus  -sticky nwes
grid .0 .pm .clear .eq    -sticky nwes

grid configure .f -sticky wes

foreach w [winfo children .] {grid configure $w -padx 3 -pady 3}

grid rowconfigure . 0 -weight 0
foreach i {1 2 3 4} { grid rowconfigure . $i -weight 1 }
foreach i {0 1 2 3} { grid columnconfigure . $i -weight 1 }

bind . 0             { appendNum 0 }
bind . <KP_Insert>   { appendNum 0 }
bind . 1             { appendNum 1 }
bind . <KP_End>      { appendNum 1 }
bind . 2             { appendNum 2 }
bind . <KP_Down>     { appendNum 2 }
bind . 3             { appendNum 3 }
bind . <KP_Next>     { appendNum 3 }
bind . 4             { appendNum 4 }
bind . <KP_Left>     { appendNum 4 }
bind . 5             { appendNum 5 }
bind . <KP_Begin>    { appendNum 5 }
bind . 6             { appendNum 6 }
bind . <KP_Right>    { appendNum 6 }
bind . 7             { appendNum 7 }
bind . <KP_Home>     { appendNum 7 }
bind . 8             { appendNum 8 }
bind . <KP_Up>       { appendNum 8 }
bind . 9             { appendNum 9 }
bind . <KP_Prior>    { appendNum 9 }
bind . +             { applyOp + }
bind . <KP_Add>      { applyOp + }
bind . -             { applyOp - }
bind . <KP_Subtract> { applyOp - }
bind . *             { applyOp * }
bind . <KP_Multiply> { applyOp * }
bind . /             { applyOp / }
bind . <KP_Divide>   { applyOp / }
bind . <Return>      { applyOp = }
bind . <KP_Enter>    { applyOp = }
bind . <BackSpace>   { clearAll }
	`)

	ir.RegisterCommand("appendNum", func(n string) {
		if afterOp {
			afterOp = false
			ir.Eval("set calcText {}")
		}
		ir.Eval("append calcText %{}", n)
	})

	ir.RegisterCommand("applyOp", func(op string) {
		if afterOp && lastOp != "=" {
			return
		}
		applyOp(op, ir)
		ir.Eval("set lastOp %{}", lastOp)
	})

	ir.RegisterCommand("clearAll", func() {
		args[0] = nil
		args[1] = nil
		afterOp = true
		lastOp = ""
		ir.Eval("set lastOp {}; set calcText 0")
	})

	ir.RegisterCommand("plusMinus", func() {
		var text string
		ir.EvalAs(&text, "set calcText")
		if len(text) == 0 || text[0] == '0' {
			return
		}

		if text[0] == '-' {
			ir.Eval("set calcText %{}", text[1:])
		} else {
			ir.Eval("set calcText -%{}", text)
		}
	})


	<-ir.Done
}
