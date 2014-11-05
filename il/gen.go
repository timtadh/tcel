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
	_, eblk := g.Stmts(node, nil, g.fn.Entry())
	eblk.Add(NewInst(Ops["EXIT"], &UNIT, &UNIT, &UNIT))
	return g.funcs, nil
}

type ilGen struct {
	syms   *table.SymbolTable
	types  *table.SymbolTable
	funcs Functions
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
		g:         g,
	}
	if g.fn != nil {
		f.StaticScope = append(g.fn.StaticScope, g.fn)
	}
	f.Scope = uint16(len(f.StaticScope))
	f.entry = f.AddNewBlock()
	g.funcs[name] = f
	return f
}

func (g *ilGen) Push() {
	g.syms.Push()
	g.types.Push()
}

func (g *ilGen) Pop() {
	if err := g.types.Pop(); err != nil {
		panic(err)
	}
	if err := g.syms.Pop(); err != nil {
		panic(err)
	}
}

func (g *ilGen) CONST(v interface{}, t types.Type) *Operand {
	return &Operand{Type: t, Value: &Constant{v}}
}

func (g *ilGen) Register(t types.Type) *Operand {
	return g.fn.NewRegister(t)
}

func (g *ilGen) Stmts(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	for i, stmt := range node.Children {
		if i + 1 == len(node.Children) {
			rslt, blk = g.Stmt(stmt, rslt, blk)
		} else {
			_, blk = g.Stmt(stmt, nil, blk)
		}
	}
	return rslt, blk
}

func (g *ilGen) Stmt(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	switch node.Label {
	case "Assign":
		return g.Assign(node, rslt, blk)
	default:
		return g.Expr(node, rslt, blk)
	}
}

func (g *ilGen) Assign(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	if rslt != nil {
		panic(fmt.Errorf("cannot propogate the result of an assign"))
	}
	expr, blk := g.Expr(node.Get(1), nil, blk)
	return g.assign(node.Get(0), expr, blk)
}

func (g *ilGen) assign(node *frontend.Node, expr *Operand, blk *Block) (*Operand, *Block) {
	if node.Label == "NAME" {
		name := g.NAME(node)
		g.syms.Put(name, expr)
		return &UNIT, blk
	} else if node.Label == "Deref" {
		var sym *Operand
		sym, blk = g.Symbol(node.Get(0), nil, blk)
		size := g.primative_size(sym)
		blk.Add(NewInst(Ops["PUT"], expr, sym, OffLen(0, size)))
		return &UNIT, blk
	} else {
		panic(fmt.Errorf("Array assignments unimplemented"))
	}
}

/*
func (g *ilGen) assignee(node *frontend.Node) (value []interface{}, i int64) {
	if node.Label == "Index" {
		item := g.Expr(node.Get(0)).([]interface{})
		spot := g.Expr(node.Get(1)).(int64)
		return item, spot
	} else {
		panic(fmt.Errorf("Unxpected node %v", node))
	}
}
*/

func (g *ilGen) NAME(node *frontend.Node) (string) {
	if node.Label != "NAME" {
		panic(fmt.Errorf("expected a NAME node : %v", node.Serialize(true)))
	}
	return node.Value.(string)
}

func (g *ilGen) Expr(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	switch node.Label {
	case "+", "-", "*", "/", "%":
		return g.ArithOp(node, rslt, blk)
	case "Negate", "Deref":
		return g.UnaryOp(node, rslt, blk)
	case "INT", "FLOAT", "STRING":
		return g.Constant(node, rslt, blk)
	case "NAME":
		return g.Symbol(node, rslt, blk)
	case "If":
		return g.If(node, rslt, blk)
	case "Func":
		return g.Function(node, rslt, blk)
	case "Call":
		return g.Call(node, rslt, blk)
	// case "Index":
		// return g.Index(node)
	case "NEW":
		return g.New(node, rslt, blk)
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
	a, blk := g.Expr(node.Get(0), nil, blk)
	if rslt == nil {
		rslt = g.Register(node.Type)
	}
	if node.Label == "Negate" {
		t := node.Get(0).Type
		zero := g.CONST(t.Empty(), t)
		blk.Add(NewInst(Ops["SUB"], zero, a, rslt))
		return rslt, blk
	} else if node.Label == "Deref" {
		size := g.primative_size(a)
		blk.Add(NewInst(Ops["GET"], a, OffLen(0, size), rslt))
		return rslt, blk
	}
	panic(fmt.Errorf("Unexpected node %v", node.Label))
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
	if sym := g.syms.Get(g.NAME(node)); sym == nil {
		panic(fmt.Errorf("Unknown name, %v", node.Serialize(true)))
	} else {
		o := sym.(*Operand)
		if rslt == nil {
			return o, blk
		}
		blk.Add(NewInst(Ops["MV"], o, &UNIT, rslt))
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
	then_last, then_blk = g.Stmts(then, rslt, then_blk)
	then_blk.J(final_blk)
	g.Pop()

	g.Push()
	var else_last *Operand
	else_last, else_blk = g.Stmts(otherwise, rslt, else_blk)
	else_blk.J(final_blk)
	g.Pop()

	if !then_last.Equals(else_last) {
		panic(fmt.Errorf("should have the same result on both branches"))
	}

	return rslt, final_blk
}

func (g *ilGen) BooleanExpr(node *frontend.Node, blk, then, otherwise *Block) (*Block) {
	switch node.Label {
	case "TRUE":
		blk.J(then)
		return blk
	case "FALSE":
		blk.J(otherwise)
		return blk
	case "<", "<=", "==", "!=", ">=", ">":
		return g.CmpOp(node, blk, then, otherwise)
	case "||":
		return g.Or(node, blk, then, otherwise)
	case "&&":
		return g.And(node, blk, then, otherwise)
	case "!":
		return g.Not(node, blk, then, otherwise)
	default:
		panic(fmt.Errorf("unexpected node %v", node))
	}
}

func (g *ilGen) CmpOp(node *frontend.Node, blk, then, otherwise *Block) (*Block) {
	a, blk := g.Expr(node.Get(0), nil, blk)
	b, blk := g.Expr(node.Get(1), nil, blk)
	var op OpCode
	switch node.Label {
	case "<": op = Ops["IFLT"]
	case "<=": op = Ops["IFLE"]
	case "==": op = Ops["IFEQ"]
	case "!=": op = Ops["IFNQ"]
	case ">=": op = Ops["IFGE"]
	case ">": op = Ops["IFGT"]
	default: panic(fmt.Errorf("unexpected node %v", node))
	}
	blk.Add(NewInst(op, a, b, Jump(then)))
	blk.J(otherwise)
	return blk
}

func (g *ilGen) Or(node *frontend.Node, blk, then, otherwise *Block) (*Block) {
	bblk := g.BooleanExpr(node.Get(1), g.fn.AddNewBlock(), then, otherwise)
	ablk := g.BooleanExpr(node.Get(0), blk, then, bblk)
	return ablk
}

func (g *ilGen) And(node *frontend.Node, blk, then, otherwise *Block) (*Block) {
	bblk := g.BooleanExpr(node.Get(1), g.fn.AddNewBlock(), then, otherwise)
	ablk := g.BooleanExpr(node.Get(0), blk, bblk, otherwise)
	return ablk
}

func (g *ilGen) Not(node *frontend.Node, blk, then, otherwise *Block) (*Block) {
	return g.BooleanExpr(node.Get(0), blk, otherwise, then)
}

func (g *ilGen) Function(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	params := node.Get(0)
	ret_type := node.Get(1)
	block := node.Get(2)

	if rslt == nil {
		rslt = g.Register(node.Type)
	}

	f := g.NewFunc(node.Type.(*types.Function))
	blk.Add(NewInst(Ops["IMM"], Call(f), &UNIT, rslt))

	g.Push()
	defer g.Pop()

	fblk := f.Entry()
	old_fn := g.fn
	g.fn = f

	defer func() {
		g.fn = old_fn
	}()

	for i, kid := range params.Children {
		t := kid.Type
		name := g.NAME(kid.Get(0))
		reg := g.Register(t)
		fblk.Add(NewInst(Ops["PRM"], Const(i), &UNIT, reg))
		g.syms.Put(name, reg)
	}

	ret, xblk := g.Stmts(block, nil, fblk)

	// here we need to do escape analysis and make closures as appropriate

	if !ret_type.Type.Equals(types.Unit) {
		xblk.Add(NewInst(Ops["RTRN"], ret, &UNIT, &UNIT))
	}

	if _, is := ret_type.Type.(*types.Function); is {
		panic(fmt.Errorf("Doesn't yet support closures sorry!\n%v", node.Serialize(true)))
	}

	return rslt, blk
}

func (g *ilGen) Call(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	g.Push()
	defer g.Pop()

	params, blk := g.Params(node.Get(1), blk)
	callee, blk := g.Expr(node.Get(0), nil, blk)

	if rslt == nil {
		rslt = g.Register(node.Type)
	}

	blk.Add(NewInst(Ops["CALL"], callee, Params(params), rslt))
	return rslt, blk
}

func (g *ilGen) Params(node *frontend.Node, blk *Block) (prms []*Operand, oblk *Block) {
	for _, kid := range node.Children {
		var prm *Operand
		prm, blk = g.Expr(kid, nil, blk)
		prms = append(prms, prm)
	}
	return prms, blk
}

func (g *ilGen) New(node *frontend.Node, rslt *Operand, blk *Block) (*Operand, *Block) {
	t := node.Get(0).Type
	if at, is := t.(*types.Array); is {
		fmt.Printf("alloc array %v\n%v\n", at, node.Serialize(true))
		panic(fmt.Errorf("cannot alloc a\n%v", node.Serialize(true)))
	} else if pt, is := t.(types.Primative); is {
		return g.new_primative(node, pt, rslt, blk)
	} else {
		panic(fmt.Errorf("cannot alloc a\n%v", node.Serialize(true)))
	}
	return rslt, blk
}

func (g *ilGen) primative_size(o *Operand) int {
	switch t := o.Type.Unboxed().(type) {
	case types.Primative:
		switch t {
		case "int": return 4
		case "float": return 4
		}
	}
	panic(fmt.Errorf("can't get the size of %v", o))
}

func (g *ilGen) new_primative(node *frontend.Node, p types.Primative, rslt *Operand, blk *Block) (*Operand, *Block) {
	size := 0
	switch p {
	case "int": size = 4
	case "float": size = 4
	default: panic(fmt.Errorf("Cannot alloc a\n%v", node.Serialize(true)))
	}
	if rslt == nil {
		rslt = g.Register(node.Type)
	}
	blk.Add(NewInst(Ops["NEW"], Const(size), &UNIT, rslt))
	return rslt, blk
}

