package checker

import (
	"fmt"
	"strings"
)

import (
	"github.com/cwru-compilers/type-check-example/frontend"
	"github.com/cwru-compilers/type-check-example/types"
	"github.com/cwru-compilers/type-check-example/table"
)

type Errors []error

func (self Errors) Error() string {
	errs := make([]string, 0, len(self))
	for _, e := range self {
		errs = append(errs, fmt.Sprintf("\"%v\"", e))
	}
	return "[" + strings.Join(errs, ", ") + "]"
}

func matches(a types.Type, ts ...types.Type) bool {
	for _, t := range ts {
		if a.Equals(t) {
			return true
		}
	}
	return false
}

func Check(node *frontend.Node) error {
	c := newChecker()
	errors := c.Stmts(node)
	if len(errors) == 0 {
		return nil
	}
	return errors
}

type checker struct {
	syms   *table.SymbolTable
	types  *table.SymbolTable
	fn     *types.Function
}

func newChecker() *checker {
	c := &checker{
		syms:  table.NewSymbolTable(),
		types: table.NewSymbolTable(),
	}
	for _, p := range types.Primatives {
		c.types.Put(string(p), p)
	}
	return c
}

func (c *checker) Push() {
	c.syms.Push()
	c.types.Push()
}

func (c *checker) Pop() {
	if err := c.types.Pop(); err != nil {
		panic(err)
	}
	if err := c.syms.Pop(); err != nil {
		panic(err)
	}
}


func (c *checker) Stmts(node *frontend.Node) (errors Errors) {
	if node.Label != "Stmts" {
		panic("expected a stmts node")
	}
	for _, stmt := range node.Children {
		errors = append(errors, c.Stmt(stmt)...)
	}
	if len(errors) == 0 {
		node.Type = types.Unit
	}
	return errors
}

func (c *checker) Stmt(node *frontend.Node) (errors Errors) {
	switch node.Label {
	case "Assign":
		return c.Assign(node)
	default:
		return c.Expr(node)
	}
	return nil
}

func (c *checker) Assign(node *frontend.Node) (errors Errors) {
	name := node.Get(0).Value.(string)
	expr := node.Get(-1)
	errors = append(errors, c.Expr(expr)...)
	if len(errors) == 0 {
		node.Get(0).Type = expr.Type
		c.syms.Put(name, expr.Type)
		node.Type = types.Unit
	}
	return errors
}

func (c *checker) Expr(node *frontend.Node) (errors Errors) {
	switch node.Label {
	case "+", "-", "*", "/", "%":
		errors = c.ArithOp(node)
	case "Negate":
		errors = c.UnaryOp(node)
	case "INT":
		node.Type = types.Int
	case "FLOAT":
		node.Type = types.Float
	case "STRING":
		node.Type = types.String
	case "NAME":
		errors = c.Symbol(node)
	case "Call":
		errors = c.Call(node)
	case "Func":
		errors = c.Function(node)
	default:
		errors = append(errors, fmt.Errorf("unexpected node %v", node))
	}
	return errors
}

func (c *checker) Call(node *frontend.Node) (errors Errors) {
	callee := node.Get(0)
	params := node.Get(1)

	err := c.Expr(callee)
	if err != nil {
		return err
	}

	param_types, err := c.Params(params)
	if err != nil {
		return err
	}

	f_type, ok := callee.Type.(*types.Function)
	if !ok {
		return append(errors, fmt.Errorf("Expected a function type got, %v", callee))
	}

	if len(param_types) != len(f_type.Parameters) {
		return append(errors, fmt.Errorf("Callee expected %v params got %v", f_type.Parameters, param_types))
	}

	for i, t := range f_type.Parameters {
		if !t.Equals(param_types[i]) {
			return append(errors, fmt.Errorf("Callee expected %v params got %v", t, param_types[i]))
		}
	}

	node.Type = f_type.Returns
	
	return errors
}

func (c *checker) Params(node *frontend.Node) (typ []types.Type, errors Errors) {
	for _, kid := range node.Children {
		err := c.Expr(kid)
		if err != nil {
			return nil, err
		}
		typ = append(typ, kid.Type)
	}
	node.Type = types.Unit
	return typ, nil
}

func (c *checker) Function(node *frontend.Node) (errors Errors) {
	params := node.Get(0)
	ret_type := node.Get(1)
	block := node.Get(2)

	c.Push()

	param_types, err := c.ParamDecls(params)
	if err != nil {
		return append(errors, err...)
	}

	return_type, err := c.Type(ret_type)
	if err != nil {
		return append(errors, err...)
	}

	f_type := &types.Function{
		Parameters: param_types,
		Returns: return_type,
	}

	if len(errors) == 0 {
		old_fn := c.fn
		c.fn = f_type

		errors = append(errors, c.Stmts(block)...)
		if len(errors) != 0 {
			return errors
		}

		c.fn = old_fn

		last := block.Get(-1)
		if !f_type.Returns.Equals(last.Type) {
			return append(errors, fmt.Errorf("Function type does not agree with last expression."))
		}

		if len(errors) == 0 {
			node.Type = f_type
		}
	}
	c.Pop()
	return errors
}

func (c *checker) Type(node *frontend.Node) (typ types.Type, errors Errors) {
	switch node.Label {
	case "TypeName":
		return c.TypeName(node)
	case "FuncType":
		return c.FuncType(node)
	}
	return nil, append(errors, fmt.Errorf("Unexpected node label %v", node))
}

func (c *checker) TypeName(node *frontend.Node) (typ types.Type, errors Errors) {
	sym := node.Get(0).Value.(string)
	if e := c.types.Get(sym); e == nil {
		errors = append(errors, fmt.Errorf("type, %v, undeclared", node.Get(0).Serialize(true)))
	} else {
		node.Type = e.(types.Type)
		node.Get(0).Type = node.Type
	}
	return node.Type, errors
}

func (c *checker) FuncType(node *frontend.Node) (typ types.Type, errors Errors) {
	params, err := c.TypeParams(node.Get(0))
	if err != nil {
		return nil, err
	}
	ret_type, err := c.Type(node.Get(1))
	if err != nil {
		return nil, err
	}
	node.Type = &types.Function{
		Parameters: params,
		Returns: ret_type,
	}
	return node.Type, errors
}

func (c *checker) TypeParams(node *frontend.Node) (typ []types.Type, errors Errors) {
	for _, kid := range node.Children {
		t, err := c.Type(kid)
		if err != nil {
			return nil, err
		}
		typ = append(typ, t)
	}
	node.Type = types.Unit
	return typ, nil
}

func (c *checker) ParamDecls(node *frontend.Node) (typ []types.Type, errors Errors) {
	for _, kid := range node.Children {
		n := kid.Get(0)
		if n.Label != "NAME" {
			return nil, append(errors, fmt.Errorf("Expected a NAME node in ParamDecls"))
		}
		name := n.Value.(string)
		t, err := c.Type(kid.Get(1))
		if err != nil {
			return nil, err
		}
		c.syms.Put(name, t)
		typ = append(typ, t)
		n.Type = t
	}
	node.Type = types.Unit
	return typ, nil
}

func (c *checker) Symbol(node *frontend.Node) (errors Errors) {
	if node.Label != "NAME" {
		return append(errors, fmt.Errorf("expected a symbol node"))
	}
	sym := node.Value.(string)
	if e := c.syms.Get(sym); e == nil {
		errors = append(errors, fmt.Errorf("symbol, %v, undeclared", node.Serialize(true)))
	} else {
		node.Type = e.(types.Type)
	}
	return errors
}

func (c *checker) ArithOp(node *frontend.Node) (errors Errors) {
	a := node.Children[0]
	b := node.Children[1]
	errors = append(errors, c.Expr(a)...)
	errors = append(errors, c.Expr(b)...)
	if len(errors) == 0 {
		if !a.Type.Equals(b.Type) {
			errors = append(errors, fmt.Errorf("a, %v, does not agree with b, %v, in types", a, b))
		}
		if a.Type.Equals(types.String) && node.Label == "+" {
			// ok
		} else if a.Type.Equals(types.Float) && node.Label == "%" {
			errors = append(errors, fmt.Errorf("type %v does not support % op", a))
		} else if !matches(a.Type, types.Int, types.Float) {
			errors = append(errors, fmt.Errorf("type %v does not support arith ops", a))
		}
	}
	if len(errors) == 0 {
		node.Type = a.Type
	}
	return errors
}

func (c *checker) UnaryOp(node *frontend.Node) (errors Errors) {
	a := node.Children[0]
	errors = append(errors, c.Expr(a)...)
	if len(errors) == 0 && !matches(a.Type, types.Int, types.Float) {
		errors = append(errors, fmt.Errorf("type %v does not support arith ops", a))
	}
	if len(errors) == 0 {
		node.Type = a.Type
	}
	return errors
}

