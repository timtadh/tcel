package checker

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/tcel/frontend"
	"github.com/timtadh/tcel/types"
	"github.com/timtadh/tcel/table"
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
	c.syms.Put("unit", types.Unit)
	c.syms.Put("print_int", &types.Function{
		Parameters: []types.Type{ types.Type(types.Int) },
		Returns: types.Unit,
	})
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
		if stmt.Type == nil {
			errors = append(errors, fmt.Errorf("Stmt not well typed : %v", stmt.Serialize(true)))
		}
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
	name := node.Get(0)
	expr := node.Get(1)
	errors = append(errors, c.Indexed(name)...)
	errors = append(errors, c.Expr(expr)...)
	if len(errors) == 0 {
		if name.Type == nil {
			sym, errors := c.NAME(name)
			if len(errors) != 0 {
				return errors
			}
			name.Type = expr.Type
			c.syms.Put(sym, expr.Type)
			node.Type = types.Unit
		} else if !name.Type.Equals(expr.Type) {
			errors = append(errors, fmt.Errorf("Assignee did not agree in types with Assinged : %v", node.Serialize(true)))
		} else {
			node.Type = types.Unit
		}
	}
	return errors
}

func (c *checker) Indexed(node *frontend.Node) (errors Errors) {
	if node.Label == "Deref" {
		errors = c.Symbol(node.Get(0))
		if len(errors) == 0 {
			node.Type = node.Get(0).Type.Unboxed()
		}
	} else if node.Label == "NAME" {
		errors = c.TryTopSymbol(node)
	} else if node.Label == "Index" {
		errors = append(errors, c.Indexed(node.Get(0))...)
		errors = append(errors, c.Indexer(node.Get(1))...)
		if node.Get(0).Label == "NAME" {
			errors = append(errors, c.Symbol(node.Get(0))...)
		}
		if t, isarr := node.Get(0).Type.(*types.Array); !isarr {
			errors = append(errors, fmt.Errorf("Expected a node of type array got %v %T %v", node.Get(0).Serialize(true), node.Get(0).Type, t))
		} else {
			node.Type = t.Base
		}
	} else {
		errors = append(errors, fmt.Errorf("unxpected node in Indexed : %v", node.Serialize(true)))
	}
	return errors
}

func (c *checker) Indexer(node *frontend.Node) (errors Errors) {
	errors = c.Expr(node)
	if len(errors) > 0 {
		return errors
	}
	if !node.Type.Equals(types.Int) {
		return append(errors, fmt.Errorf("Expected a node of type int as the index got %v", node.Serialize(true)))
	}
	return nil
}

func (c *checker) NAME(node *frontend.Node) (name string, errors Errors) {
	if node.Label != "NAME" {
		return "", append(errors, fmt.Errorf("expected a NAME node : %v", node.Serialize(true)))
	}
	return node.Value.(string), nil
}

func (c *checker) Expr(node *frontend.Node) (errors Errors) {
	switch node.Label {
	case "+", "-", "*", "/", "%":
		errors = c.ArithOp(node)
	case "Negate", "Deref":
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
	case "Index":
		errors = c.Index(node)
	case "Func":
		errors = c.Function(node)
	case "If":
		errors = c.If(node)
	case "NEW":
		errors = c.New(node)
	default:
		errors = append(errors, fmt.Errorf("unexpected node %v", node))
	}
	return errors
}

func (c *checker) New(node *frontend.Node) (errors Errors) {
	new_type, err := c.Type(node.Get(0))
	if err != nil {
		return append(errors, err...)
	}
	if _, ok := new_type.(*types.Function); ok {
		return append(errors, fmt.Errorf("Cannot construct a function with new %v", node.Serialize(true)))
	}
	if _, ok := new_type.(*types.Array); ok {
		node.Type = new_type
	} else {
		node.Type = &types.Box{new_type}
	}
	return errors
}

func (c *checker) BooleanExpr(node *frontend.Node) (errors Errors) {
	switch node.Label {
	case "TRUE", "FALSE":
		errors = c.BooleanConstant(node)
	case "<", "<=", "==", "!=", ">=", ">":
		errors = c.CmpOp(node)
	case "||":
		errors = c.Or(node)
	case "&&":
		errors = c.And(node)
	case "!":
		errors = c.Not(node)
	default:
		errors = append(errors, fmt.Errorf("unexpected node %v", node))
	}
	return errors
}

func (c *checker) Index(node *frontend.Node) (errors Errors) {
	indexed := node.Get(0)
	index := node.Get(1)

	err := c.Expr(indexed)
	if err != nil {
		return err
	}

	err = c.Expr(index)
	if err != nil {
		return err
	}

	a_type, ok := indexed.Type.(*types.Array)
	if !ok {
		return append(errors, fmt.Errorf("Expected a array type got, %v", indexed.Serialize(true)))
	}

	if !index.Type.Equals(types.Int) {
		return append(errors, fmt.Errorf("Array index expected int got %v", index.Serialize(true)))
	}

	node.Type = a_type.Base

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
	defer c.Pop()

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

		c.syms.Put("self", f_type)
		errors = append(errors, c.Stmts(block)...)
		if len(errors) != 0 {
			return errors
		}

		c.fn = old_fn

		last := block.Get(-1)
		if !f_type.Returns.Equals(last.Type) {
			return append(errors,
				fmt.Errorf(
					"Function type, %v, does not agree with last expression, %v, at %v",
					f_type,
					last.Type,
					last.Serialize(true),
				))
		}

		if len(errors) == 0 {
			node.Type = f_type
		}
	}
	return errors
}

func (c *checker) If(node *frontend.Node) (errors Errors) {
	condition := node.Get(0)
	then := node.Get(1)
	otherwise := node.Get(2)

	errors = append(errors, c.BooleanExpr(condition)...)
	c.Push()
	errors = append(errors, c.Stmts(then)...)
	c.Pop()
	c.Push()
	errors = append(errors, c.Stmts(otherwise)...)
	c.Pop()


	if len(errors) != 0 {
		return errors
	}

	then.Type = then.Get(-1).Type
	otherwise.Type = otherwise.Get(-1).Type

	if !then.Type.Equals(otherwise.Type) {
		return append(errors, fmt.Errorf("Branches of if expression do not agree in types. %v", node.Serialize(true)))
	}

	node.Type = then.Type

	return errors
}

func (c *checker) Type(node *frontend.Node) (typ types.Type, errors Errors) {
	switch node.Label {
	case "TypeName":
		return c.TypeName(node)
	case "FuncType":
		return c.FuncType(node)
	case "ArrayType":
		return c.ArrayType(node)
	case "BoxType":
		return c.BoxType(node)
	}
	return nil, append(errors, fmt.Errorf("Unexpected node label %v", node))
}

func (c *checker) TypeName(node *frontend.Node) (typ types.Type, errors Errors) {
	sym, errors := c.NAME(node.Get(0))
	if len(errors) > 0 {
		return nil, errors
	}
	if e := c.types.Get(sym); e == nil {
		errors = append(errors, fmt.Errorf("type, %v, undeclared", node.Get(0).Serialize(true)))
	} else {
		node.Type = e.(types.Type)
		node.Get(0).Type = node.Type
	}
	return node.Type, errors
}


func (c *checker) BoxType(node *frontend.Node) (typ types.Type, errors Errors) {
	t, errors := c.Type(node.Get(0))
	if len(errors) == 0 {
		node.Type = &types.Box{t}
		return node.Type, errors
	}
	return nil, errors
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

func (c *checker) ArrayType(node *frontend.Node) (typ types.Type, errors Errors) {
	base, err := c.Type(node.Get(0))
	if err != nil {
		return nil, err
	}
	if len(node.Children) > 1 {
		err = c.Expr(node.Get(1))
		if err != nil {
			return nil, err
		}
		if !types.Int.Equals(node.Get(1).Type) {
			return nil, append(errors, fmt.Errorf("Expected an integer size got %v %v", node.Get(1).Type, node.Serialize(true)))
		}
	}
	node.Type = &types.Array{
		Base: base,
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
		name, err := c.NAME(n)
		if err != nil {
			return nil, err
		}
		t, err := c.Type(kid.Get(1))
		if err != nil {
			return nil, err
		}
		c.syms.Put(name, t)
		typ = append(typ, t)
		n.Type = t
		kid.Type = t
	}
	node.Type = types.Unit
	return typ, nil
}

func (c *checker) TryTopSymbol(node *frontend.Node) (errors Errors) {
	sym, errors := c.NAME(node)
	if len(errors) > 0 {
		return errors
	}
	if c.syms.TopHas(sym) {
		e := c.syms.Get(sym)
		node.Type = e.(types.Type)
	}
	return errors
}

func (c *checker) TrySymbol(node *frontend.Node) (errors Errors) {
	sym, errors := c.NAME(node)
	if len(errors) > 0 {
		return errors
	}
	if e := c.syms.Get(sym); e != nil {
		node.Type = e.(types.Type)
	}
	return errors
}

func (c *checker) Symbol(node *frontend.Node) (errors Errors) {
	errors = c.TrySymbol(node)
	if len(errors) > 0 {
		return errors
	}
	if node.Type == nil {
		errors = append(errors, fmt.Errorf("symbol, %v, undeclared", node.Serialize(true)))
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
	if node.Label == "Negate" {
		if len(errors) == 0 && !matches(a.Type, types.Int, types.Float) {
			errors = append(errors, fmt.Errorf("type %v does not support arith ops", a))
		}
		if len(errors) == 0 {
			node.Type = a.Type
		}
	} else if node.Label == "Deref" {
		if box, is := a.Type.(*types.Box); !is {
			errors = append(errors, fmt.Errorf("type %v does not support deref ops", a))
		} else if len(errors) == 0 {
			node.Type = box.Boxed
		}
	} else {
		return append(errors, fmt.Errorf("Unexpected node %v", node.Serialize(true)))
	}
	return errors
}

func (c *checker) Or(node *frontend.Node) (errors Errors) {
	return c.AndOr(node)
}

func (c *checker) And(node *frontend.Node) (errors Errors) {
	return c.AndOr(node)
}

func (c *checker) AndOr(node *frontend.Node) (errors Errors) {
	a := node.Children[0]
	b := node.Children[1]
	errors = append(errors, c.BooleanExpr(a)...)
	errors = append(errors, c.BooleanExpr(b)...)
	if len(errors) == 0 {
		if !a.Type.Equals(b.Type) {
			errors = append(errors, fmt.Errorf("a, %v, does not agree with b, %v, in types", a, b))
		}
		if !matches(a.Type, types.Boolean) {
			errors = append(errors, fmt.Errorf("type %v does not support boolean ops", a))
		}
	}
	if len(errors) == 0 {
		node.Type = types.Boolean
	}
	return errors
}

func (c *checker) Not(node *frontend.Node) (errors Errors) {
	a := node.Children[0]
	errors = append(errors, c.BooleanExpr(a)...)
	if len(errors) == 0 {
		if !matches(a.Type, types.Boolean) {
			errors = append(errors, fmt.Errorf("type %v does not support boolean ops", a))
		}
	}
	if len(errors) == 0 {
		node.Type = types.Boolean
	}
	return errors
}

func (c *checker) CmpOp(node *frontend.Node) (errors Errors) {
	a := node.Children[0]
	b := node.Children[1]
	errors = append(errors, c.Expr(a)...)
	errors = append(errors, c.Expr(b)...)
	if len(errors) == 0 {
		if !a.Type.Equals(b.Type) {
			errors = append(errors, fmt.Errorf("a, %v, does not agree with b, %v, in types", a, b))
		}
		if !matches(a.Type, types.Int, types.Float, types.String) {
			errors = append(errors, fmt.Errorf("type %v does not support boolean comparison ops", a))
		}
	}
	if len(errors) == 0 {
		node.Type = types.Boolean
	}
	return errors
}

func (c *checker) BooleanConstant(node *frontend.Node) (errors Errors) {
	node.Type = types.Boolean
	return errors
}


