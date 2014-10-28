package il

import (
	"fmt"
)

import (
	"github.com/timtadh/tcel/checker"
	"github.com/timtadh/tcel/frontend"
	"github.com/timtadh/tcel/table"
	"github.com/timtadh/tcel/types"
)

func Generate(node *frontend.Node) (funcs Functions, err error) {
	if err := checker.Check(node); err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			funcs = nil
			err = fmt.Errorf("%v", e)
		}
	}()
	g := newIlGen()
	g.Stmts(node, g.fn.Entry())
	return g.funcs, nil
}

type ilGen struct {
	syms   *table.SymbolTable
	types  *table.SymbolTable
	funcs Functions
	func_depth uint16
	fn *Func
}

func newIlGen() *ilGen {
	funcs := make(Functions)
	g := &ilGen{
		funcs: funcs,
		syms:  table.NewSymbolTable(),
		types: table.NewSymbolTable(),
	}
	for _, p := range types.Primatives {
		g.types.Put(string(p), p)
	}
	g.syms.Put("unit", types.Unit)
	g.fn = g.NewFunc(
		&types.Function{
			Parameters: []types.Type{},
			Returns: types.Unit,
		},
	)
	return g
}

func (g *ilGen) NewFunc(t *types.Function) *Func {
	var name string
	if len(g.funcs) == 0 {
		name = "main"
	} else {
		name = fmt.Sprintf("fn-%d", len(g.funcs))
	}
	f := &Func{
		Name:      name,
		Type:      t,
		Blocks:    make(map[string]*Block),
		BlockList: make([]*Block, 0, 10),
		Scope:     g.func_depth,
		g:         g,
	}
	if g.fn != nil {
		f.StaticScope = append(g.fn.StaticScope, g.fn)
	}
	f.entry = f.AddNewBlock()
	g.funcs[name] = f
	return f
}

func (g *ilGen) CONST(v interface{}, t types.Type) *Operand {
	return &Operand{Type: t, Value: &Constant{v}}
}

func (g *ilGen) Register(t types.Type) *Operand {
	return g.fn.NewRegister(t)
}

func (g *ilGen) Stmts(node *frontend.Node, blk *Block) (*Operand, *Block) {
	var last *Operand = nil
	for _, stmt := range node.Children {
		last, blk = g.Stmt(stmt, blk)
	}
	return last, blk
}

func (g *ilGen) Stmt(node *frontend.Node, blk *Block) (*Operand, *Block) {
	var last *Operand = nil
	switch node.Label {
	case "Assign":
		panic("wizard")
		// return g.Assign(node)
	default:
		last, blk = g.Expr(node, nil, blk)
	}
	return last, blk
}

func (g *ilGen) Expr(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	switch node.Label {
	case "+", "-", "*", "/", "%":
		return g.ArithOp(node, rslt, blk)
	case "Negate":
		return g.UnaryOp(node, rslt, blk)
	case "INT", "FLOAT", "STRING":
		return g.Constant(node, rslt, blk)
	case "NAME":
		return g.Symbol(node, rslt, blk)
	case "If":
		return g.If(node, rslt, blk)
	// case "Call":
		// return g.Call(node)
	// case "Index":
		// return g.Index(node)
	// case "Func":
		// return (*function)(node)
	// case "Params":
		// return g.Params(node)
	// case "NEW":
		// return g.New(node)
	default:
		panic(fmt.Errorf("unexpected node %v", node))
	}
}

func (g *ilGen) ArithOp(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	a, blk := g.Expr(node.Get(0), nil, blk)
	b, blk := g.Expr(node.Get(1), nil, blk)
	var op OpCode
	switch node.Label {
	case "+": op = Ops["ADD"]
	case "-": op = Ops["SUB"]
	case "*": op = Ops["MUL"]
	case "/": op = Ops["DIV"]
	case "%": op = Ops["MOD"]
	default: panic(fmt.Errorf("Unexpected node %v", node.Label))
	}
	if rslt == nil {
		rslt = g.Register(node.Type)
	}
	blk.Add(NewInst(op, a, b, rslt))
	return rslt, blk
}

func (g *ilGen) UnaryOp(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	if node.Label != "Negate" {
		panic(fmt.Errorf("Unexpected node %v", node.Label))
	}
	t := node.Get(0).Type
	zero := g.CONST(t.Empty(), t)
	b, blk := g.Expr(node.Get(0), nil, blk)
	if rslt == nil {
		rslt = g.Register(node.Type)
	}
	blk.Add(NewInst(Ops["SUB"], zero, b, rslt))
	return rslt, blk
}


func (g *ilGen) Constant(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	c := g.CONST(node.Value, node.Type)
	if rslt == nil {
		return c, blk
	}
	blk.Add(NewInst(Ops["IMM"], c, &UNIT, rslt))
	return rslt, blk
}

func (g *ilGen) Symbol(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	if sym := g.syms.Get(node.Value.(string)); sym == nil {
		panic(fmt.Errorf("Unknown name, %v", node.Serialize(true)))
	} else {
		o := sym.(*Operand)
		if rslt == nil {
			return o, blk
		}
		blk.Add(NewInst(Ops["IMM"], o, &UNIT, rslt))
		return rslt, blk
	}
}

func (g *ilGen) If(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	then_blk := g.fn.AddNewBlock()
	else_blk := g.fn.AddNewBlock()
	final_blk := g.fn.AddNewBlock()

	condition := node.Get(0)
	then := node.Get(1)
	otherwise := node.Get(2)

	blk = g.BooleanExpr(condition, blk, then_blk, else_blk)

	if rslt == nil {
		rslt = g.Register(node.Type)
	}

	g.Push()
	var then_last *Operand
	then_last, then_blk = g.Stmts(then, then_blk)
	then_blk.Add(NewInst(Ops["MV"], then_last, &UNIT, rslt))
	then_blk.J(final_blk)
	g.Pop()

	g.Push()
	var else_last *Operand
	else_last, else_blk = g.Stmts(otherwise, else_blk)
	else_blk.Add(NewInst(Ops["MV"], else_last, &UNIT, rslt))
	else_blk.J(final_blk)
	g.Pop()

	return rslt, final_blk
}

