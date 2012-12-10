package main

import "github.com/nsf/gothic"
import "math/big"

type calc struct {
	*gothic.Interpreter
	args [2]*big.Int
	lastOp string
	afterOp bool
}

func (c *calc) TCL_ApplyOp(op string) {
	if c.afterOp && c.lastOp != "=" {
		return
	}

	var num string
	c.EvalAs(&num, "set calcText")
	if c.args[0] == nil {
		if op != "=" {
			c.args[0] = big.NewInt(0)
			c.args[0].SetString(num, 10)
		}
	} else {
		c.args[1] = big.NewInt(0)
		c.args[1].SetString(num, 10)
	}

	c.afterOp = true

	if c.args[1] == nil {
		c.lastOp = op
		c.Eval("set lastOp %{}", c.lastOp)
		return
	}

	switch c.lastOp {
	case "+":
		c.args[0] = c.args[0].Add(c.args[0], c.args[1])
	case "-":
		c.args[0] = c.args[0].Sub(c.args[0], c.args[1])
	case "/":
		c.args[0] = c.args[0].Div(c.args[0], c.args[1])
	case "*":
		c.args[0] = c.args[0].Mul(c.args[0], c.args[1])
	}

	c.lastOp = op
	c.args[1] = nil

	c.Eval("set lastOp %{}", c.lastOp)
	c.Eval("set calcText %{}", c.args[0])
	if op == "=" {
		c.args[0] = nil
	}
}

func (c *calc) TCL_AppendNum(n string) {
	if c.afterOp {
		c.afterOp = false
		c.Eval("set calcText {}")
	}
	c.Eval("append calcText %{}", n)
}

func (c *calc) TCL_ClearAll() {
	c.args[0] = nil
	c.args[1] = nil
	c.afterOp = true
	c.lastOp = ""
	c.Eval("set lastOp {}; set calcText 0")
}

func (c *calc) TCL_PlusMinus() {
	var text string
	c.EvalAs(&text, "set calcText")
	if len(text) == 0 || text[0] == '0' {
		return
	}

	if text[0] == '-' {
		c.Eval("set calcText %{}", text[1:])
	} else {
		c.Eval("set calcText -%{}", text)
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
grid columnconfigure .f 1 -weight 1

ttk::button .0 -text 0 -command { go::AppendNum 0 }
ttk::button .1 -text 1 -command { go::AppendNum 1 }
ttk::button .2 -text 2 -command { go::AppendNum 2 }
ttk::button .3 -text 3 -command { go::AppendNum 3 }
ttk::button .4 -text 4 -command { go::AppendNum 4 }
ttk::button .5 -text 5 -command { go::AppendNum 5 }
ttk::button .6 -text 6 -command { go::AppendNum 6 }
ttk::button .7 -text 7 -command { go::AppendNum 7 }
ttk::button .8 -text 8 -command { go::AppendNum 8 }
ttk::button .9 -text 9 -command { go::AppendNum 9 }
ttk::button .pm    -text +/- -command go::PlusMinus
ttk::button .clear -text C -command go::ClearAll
ttk::button .eq    -text = -command { go::ApplyOp = }
ttk::button .plus  -text + -command { go::ApplyOp + }
ttk::button .minus -text - -command { go::ApplyOp - }
ttk::button .mul   -text * -command { go::ApplyOp * }
ttk::button .div   -text / -command { go::ApplyOp / }

grid .f -   -      .div   -sticky nwes
grid .7 .8  .9     .mul   -sticky nwes
grid .4 .5  .6     .minus -sticky nwes
grid .1 .2  .3     .plus  -sticky nwes
grid .0 .pm .clear .eq    -sticky nwes

grid configure .f -sticky we

foreach w [winfo children .] {grid configure $w -padx 3 -pady 3}

foreach i {1 2 3 4} { grid rowconfigure . $i -weight 1 }
foreach i {0 1 2 3} { grid columnconfigure . $i -weight 1 }

bind . 0             { go::AppendNum 0 }
bind . <KP_Insert>   { go::AppendNum 0 }
bind . 1             { go::AppendNum 1 }
bind . <KP_End>      { go::AppendNum 1 }
bind . 2             { go::AppendNum 2 }
bind . <KP_Down>     { go::AppendNum 2 }
bind . 3             { go::AppendNum 3 }
bind . <KP_Next>     { go::AppendNum 3 }
bind . 4             { go::AppendNum 4 }
bind . <KP_Left>     { go::AppendNum 4 }
bind . 5             { go::AppendNum 5 }
bind . <KP_Begin>    { go::AppendNum 5 }
bind . 6             { go::AppendNum 6 }
bind . <KP_Right>    { go::AppendNum 6 }
bind . 7             { go::AppendNum 7 }
bind . <KP_Home>     { go::AppendNum 7 }
bind . 8             { go::AppendNum 8 }
bind . <KP_Up>       { go::AppendNum 8 }
bind . 9             { go::AppendNum 9 }
bind . <KP_Prior>    { go::AppendNum 9 }
bind . +             { go::ApplyOp + }
bind . <KP_Add>      { go::ApplyOp + }
bind . -             { go::ApplyOp - }
bind . <KP_Subtract> { go::ApplyOp - }
bind . *             { go::ApplyOp * }
bind . <KP_Multiply> { go::ApplyOp * }
bind . /             { go::ApplyOp / }
bind . <KP_Divide>   { go::ApplyOp / }
bind . <Return>      { go::ApplyOp = }
bind . <KP_Enter>    { go::ApplyOp = }
bind . <BackSpace>   { go::ClearAll }
	`)
	ir.RegisterCommands("go", &calc{
		Interpreter: ir,
		afterOp: true,
	})
	<-ir.Done
}
