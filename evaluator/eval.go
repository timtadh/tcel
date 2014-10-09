package evaluator

import (
	"fmt"
)

import (
	"github.com/cwru-compilers/type-check-example/frontend"
	"github.com/cwru-compilers/type-check-example/types"
	"github.com/cwru-compilers/type-check-example/table"
)

func Evaluate(node *frontend.Node) (values []interface{}, err error) {
	/*defer func() {
		if e := recover(); e != nil {
			values = nil
			err = fmt.Errorf("%v", e)
		}
	}()*/
	e := newEvaluator()
	return e.Stmts(node), nil
}

func Partial() *Evaluator {
	return newEvaluator()
}

type Parameterized interface {
	ParamNames() []string
	FnType() *types.Function
}

type Evaluator struct {
	syms   *table.SymbolTable
	types  *table.SymbolTable
	fn     *types.Function
}

type function frontend.Node

func (self *function) FnType() *types.Function {
	return (*frontend.Node)(self).Type.(*types.Function)
}

func (self *function) ParamNames() []string {
	node := (*frontend.Node)(self)
	params := node.Get(0)
	param_names := make([]string, 0, len(params.Children))
	for _, p := range params.Children {
		param_name := p.Get(0).Value.(string)
		param_names = append(param_names, param_name)
	}
	return param_names
}

func (self *function) String() string {
	return fmt.Sprintf("<function %v>", (*frontend.Node)(self))
}



type closure struct {
	fn *function
	e *Evaluator
}

func (self *closure) FnType() *types.Function {
	return self.fn.FnType()
}

func (self *closure) ParamNames() []string {
	return self.fn.ParamNames()
}

func (self *closure) String() string {
	return fmt.Sprintf("<closure %v %v>", self.fn, self.e.syms)
}

func newEvaluator() *Evaluator {
	e := &Evaluator{
		syms:  table.NewSymbolTable(),
		types: table.NewSymbolTable(),
	}
	for _, p := range types.Primatives {
		e.types.Put(string(p), p)
	}
	return e
}

func (e *Evaluator) Clone() *Evaluator {
	return &Evaluator{
		syms: table.Copy(e.syms.Capture()),
		types: table.Copy(e.types.Capture()),
		fn: e.fn,
	}
}

func (e *Evaluator) Push() {
	e.syms.Push()
	e.types.Push()
}

func (e *Evaluator) Pop() {
	if err := e.types.Pop(); err != nil {
		panic(err)
	}
	if err := e.syms.Pop(); err != nil {
		panic(err)
	}
}


func (e *Evaluator) Stmts(node *frontend.Node) (values []interface{}) {
	for _, stmt := range node.Children {
		values = append(values, e.Stmt(stmt))
	}
	return values
}

func (e *Evaluator) Stmt(node *frontend.Node) (value interface{}) {
	switch node.Label {
	case "Assign":
		return e.Assign(node)
	default:
		return e.Expr(node)
	}
}

func (e *Evaluator) Assign(node *frontend.Node) (value interface{}) {
	name := node.Get(0).Value.(string)
	expr := e.Expr(node.Get(1))
	e.syms.Put(name, expr)
	return types.Unit
}

func (e *Evaluator) Expr(node *frontend.Node) (value interface{}) {
	switch node.Label {
	case "+", "-", "*", "/", "%":
		return e.ArithOp(node)
	case "Negate":
		return e.UnaryOp(node)
	case "INT":
		return node.Value.(int64)
	case "FLOAT":
		return node.Value.(float64)
	case "STRING":
		return node.Value.(string)
	case "NAME":
		if sym := e.syms.Get(node.Value.(string)); sym == nil {
			panic(fmt.Errorf("Unknown name, %v", node.Serialize(true)))
		} else {
			return sym
		}
	case "Call":
		return e.Call(node)
	case "Func":
		return (*function)(node)
	case "Params":
		return e.Params(node)
	default:
		panic(fmt.Errorf("unexpected node %v", node))
	}
}

func (e *Evaluator) Params(node *frontend.Node) (values []interface{}) {
	for _, expr := range node.Children {
		values = append(values, e.Expr(expr))
	}
	return values
}

func (e *Evaluator) Call(node *frontend.Node) (value interface{}) {
	e.Push()
	defer e.Pop()
	callee := e.Expr(node.Get(0)).(Parameterized)
	params := e.Expr(node.Get(1)).([]interface{})
	var fne *Evaluator
	var callee_stmts *frontend.Node
	if closed, isclosure := callee.(*closure); isclosure {
		fne = closed.e
		callee_stmts = (*frontend.Node)(closed.fn).Get(2)
	} else if fn, isfn := callee.(*function); isfn {
		fne = e
		callee_stmts = (*frontend.Node)(fn).Get(2)
	} else {
		panic("something besides a function or a closure")
	}
	for i, param_name := range callee.ParamNames() {
		fne.syms.Put(param_name, params[i])
	}
	fne.syms.Put("self", callee)
	values := fne.Stmts(callee_stmts)
	ret := values[len(values)-1]
	if _, retfn := callee.FnType().Returns.(*types.Function); retfn {
		ret := &closure{(ret.(*function)), fne.Clone()}
		return ret
	}
	return ret
}

func (e *Evaluator) ArithOp(node *frontend.Node) (value interface{}) {
	a := e.Expr(node.Get(0))
	b := e.Expr(node.Get(1))
	switch node.Get(0).Type.String() {
	case "int": return e.IntArithOp(node.Label, a.(int64), b.(int64))
	case "float": return e.FloatArithOp(node.Label, a.(float64), b.(float64))
	case "string": return e.StringArithOp(node.Label, a.(string), b.(string))
	}
	panic(fmt.Errorf("unexpected node type in arith op %v", node))
}

func (e *Evaluator) UnaryOp(node *frontend.Node) (value interface{}) {
	a := e.Expr(node.Get(0))
	if node.Label == "Negate" {
		switch node.Get(0).Type.String() {
		case "int": return - a.(int64)
		case "float": return - a.(float64)
		}
	}
	panic(fmt.Errorf("unexpected node type in arith op %v", node))
}

func (e *Evaluator) IntArithOp(op string, a, b int64) (int64) {
	switch op {
	case "+": return a + b
	case "-": return a - b
	case "*": return a * b
	case "/":
		if b == 0 {
			panic(fmt.Errorf("Divide by 0"))
		}
		return a / b
	case "%": 
		if b == 0 {
			panic(fmt.Errorf("Divide by 0"))
		}
		return a % b
	}
	panic(fmt.Errorf("Unsupported op %v for ints", op))
}

func (e *Evaluator) FloatArithOp(op string, a, b float64) (float64) {
	switch op {
	case "+": return a + b
	case "-": return a - b
	case "*": return a * b
	case "/":
		if b == 0 {
			panic(fmt.Errorf("Divide by 0"))
		}
		return a / b
	}
	panic(fmt.Errorf("Unsupported op %v for floats", op))
}

func (e *Evaluator) StringArithOp(op string, a, b string) (string) {
	switch op {
	case "+": return a + b
	}
	panic(fmt.Errorf("Unsupported op %v for strings", op))
}
