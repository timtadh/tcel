package frontend

import (
	"fmt"
)

/*

Stmts -> Stmt Stmts
       | Stmt

Stmt -> Assign
      | Expr

Assign -> NAME = Expr

Expr -> Term Expr'
Expr' -> + Term Expr'
       | - Term Expr'
       | e

Term -> Unary Expr'
Term' -> * Unary Term'
       | / Unary Term'
       | % Unary Term'
       | e

Unary -> PostUnary
       | - PostUnary

PostUnary -> Factor Applies
           | Factor

Applies -> ( Params ) Applies'
Applies' -> ( Params ) Applies'
          | e

Params -> Expr Params'
        | e

Params' -> , Expr Params'
         | e

Factor -> NAME
        | INT
        | FLOAT
        | STRING
        | Function
        | ( Expr )

Function -> FN ( ParamDecls ) Type { Stmts }

ParamDecls -> NAME Type ParamDecls'
            | e

ParamDecls' -> , NAME Type ParamDecls'
             | e

Type -> NAME
      | FN ( TypeParams ) Type

TypeParams -> Type TypeParams'
            | e

TypeParams' -> , Type TypeParms'
             | e

*/

func Parse(tokens []*Token) (root *Node, err error) {

	type Consumer func(i int) (int, *Node, error)
	var (
		Stmts, Stmt, Assign, Expr, Expr_, Term, Term_,
		Unary, PostUnary, Factor, Applies, Applies_, Params, Params_,
		Function, ParamDecls, ParamDecls_, Type, TypeParams, TypeParams_ Consumer
		Epsilon func(*Node) Consumer
		Consume func(string) Consumer
		Concat func(...Consumer) func(func(...*Node)(*Node, error)) Consumer
		Alt func(Consumer, Consumer) Consumer
	)

	collapse := func(subtree, extra *Node) *Node {
		if extra == nil {
			return subtree
		}
		return extra.Get(0).AddKid(subtree).AddKid(extra.Get(1))
	}

	swing := func (nodes ...*Node) (*Node, error) {
		n := NewNode("T").AddKid(nodes[0]).AddKid(collapse(nodes[1], nodes[2]))
		return n, nil
	}

	Stmts = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Stmt, Stmts)(func (nodes ...*Node) (*Node, error) {
				stmts := NewNode("Stmts").AddKid(nodes[0])
				stmts.Children = append(stmts.Children, nodes[1].Children...)
				return stmts, nil
			}),
			Concat(Stmt)(func (nodes ...*Node) (*Node, error) {
				stmts := NewNode("Stmts").AddKid(nodes[0])
				return stmts, nil
			}),
		)(i)
	}

	Stmt = func(i int) (int, *Node, error) {
		return Alt(Assign, Expr)(i)
	}

	Assign = func(i int) (int, *Node, error) {
		return Concat(Consume("NAME"), Consume("="), Expr)(
			func (nodes ...*Node) (*Node, error) {
				stmts := NewNode("Assign").AddKid(nodes[0]).AddKid(nodes[2])
				return stmts, nil
			})(i)
	}

	Expr = func(i int) (int, *Node, error) {
		return Concat(Term, Expr_)(
			func (nodes ...*Node) (*Node, error) {
				return collapse(nodes[0], nodes[1]), nil
			})(i)
	}

	Expr_ = func(i int) (int, *Node, error) {
		return Alt(
			Alt(
				Concat(Consume("+"), Term, Expr_)(swing),
				Concat(Consume("-"), Term, Expr_)(swing),
			),
			Epsilon(nil),
		)(i)
	}

	Term = func(i int) (int, *Node, error) {
		return Concat(Unary, Term_)(
			func (nodes ...*Node) (*Node, error) {
				return collapse(nodes[0], nodes[1]), nil
			})(i)
	}

	Term_ = func(i int) (int, *Node, error) {
		return Alt(
			Alt(
				Concat(Consume("*"), Unary, Term_)(swing),
				Concat(Consume("/"), Unary, Term_)(swing),
			),
			Alt(
				Concat(Consume("%"), Unary, Term_)(swing),
				Epsilon(nil),
			),
		)(i)
	}

	Unary = func(i int) (int, *Node, error) {
		return Alt(
			PostUnary,
			Concat(Consume("-"), PostUnary)(func (nodes ...*Node) (*Node, error) {
				nodes[0].Label = "Negate"
				return nodes[0].AddKid(nodes[1]), nil
			}),
		)(i)
	}

	PostUnary = func(i int) (int, *Node, error) {
		return Alt(
			Factor,
			Concat(Factor, Applies)(
				func (nodes ...*Node) (*Node, error) {
					return nodes[1].AddLeftMostKid(nodes[0], "Call"), nil
				}),
		)(i)
	}

	aply := func(nodes ...*Node) *Node {
		if nodes[3] == nil {
			return NewNode("Call").AddKid(nodes[1])
		}
		root := nodes[3]
		root.AddLeftMostKid(NewNode("Call").AddKid(nodes[1]), "Call")
		return root
	}

	Applies = func(i int) (int, *Node, error) {
		return Concat(Consume("("), Params, Consume(")"), Applies_)(
			func (nodes ...*Node) (*Node, error) {
				return aply(nodes...), nil
			})(i)
	}

	Applies_ = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Consume("("), Params, Consume(")"), Applies_)(
				func (nodes ...*Node) (*Node, error) {
					return aply(nodes...), nil
				}),
			Epsilon(nil),
		)(i)
	}

	Params = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Expr, Params_)(
				func (nodes ...*Node) (*Node, error) {
					params := NewNode("Params").AddKid(nodes[0])
					if nodes[1] != nil {
						params.Children = append(params.Children, nodes[1].Children...)
					}
					return params, nil
				}),
			Epsilon(NewNode("Params")),
		)(i)
	}

	Params_ = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Consume(","), Expr, Params_)(
				func (nodes ...*Node) (*Node, error) {
					params := NewNode("Params").AddKid(nodes[1])
					if nodes[2] != nil {
						params.Children = append(params.Children, nodes[2].Children...)
					}
					return params, nil
				}),
			Epsilon(nil),
		)(i)
	}

	Factor = func(i int) (int, *Node, error) {
		return Alt(
			Alt(
				Consume("NAME"),
				Consume("INT"),
			),
			Alt(
				Alt(
					Consume("FLOAT"),
					Consume("STRING"),
				),
				Alt(
					Function,
					Concat(Consume("("), Expr, Consume(")"))(
						func (nodes ...*Node) (*Node, error) {
							return nodes[1], nil
						}),
				),
			),
		)(i)
	}

	Function = func(i int) (int, *Node, error) {
		return Concat(
			Consume("FN"), Consume("("), ParamDecls, Consume(")"), Type,
			Consume("{"), Stmts, Consume("}"))(
			func (nodes ...*Node) (*Node, error) {
				n := NewNode("Func").AddKid(nodes[2]).AddKid(nodes[4]).AddKid(nodes[6])
				return n, nil
			})(i)
	}

	ParamDecls = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Consume("NAME"), Type, ParamDecls_)(
				func (nodes ...*Node) (*Node, error) {
					params := NewNode("ParamDecls").AddKid(
						NewNode("ParamDecl").AddKid(nodes[0]).AddKid(nodes[1]))
					if nodes[2] != nil {
						params.Children = append(params.Children, nodes[2].Children...)
					}
					return params, nil
				}),
			Epsilon(NewNode("ParamDecls")),
		)(i)
	}

	ParamDecls_ = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Consume(","), Consume("NAME"), Type, ParamDecls_)(
				func (nodes ...*Node) (*Node, error) {
					params := NewNode("ParamDecls").AddKid(
						NewNode("ParamDecl").AddKid(nodes[1]).AddKid(nodes[2]))
					if nodes[3] != nil {
						params.Children = append(params.Children, nodes[3].Children...)
					}
					return params, nil
				}),
			Epsilon(nil),
		)(i)
	}

	Type = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Consume("NAME"))(
				func (nodes ...*Node) (*Node, error) {
					return NewNode("TypeName").AddKid(nodes[0]), nil
				}),
			Concat(Consume("FN"), Consume("("), TypeParams, Consume(")"), Type)(
				func (nodes ...*Node) (*Node, error) {
					n := NewNode("FuncType").AddKid(nodes[2]).AddKid(nodes[4])
					return n, nil
				}),
		)(i)
	}

	TypeParams = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Type, TypeParams_)(
				func (nodes ...*Node) (*Node, error) {
					params := NewNode("TypeParams").AddKid(nodes[0])
					if nodes[1] != nil {
						params.Children = append(params.Children, nodes[1].Children...)
					}
					return params, nil
				}),
			Epsilon(NewNode("TypeParams")),
		)(i)
	}

	TypeParams_ = func(i int) (int, *Node, error) {
		return Alt(
			Concat(Consume(","), Type, TypeParams_)(
				func (nodes ...*Node) (*Node, error) {
					params := NewNode("TypeParams").AddKid(nodes[1])
					if nodes[2] != nil {
						params.Children = append(params.Children, nodes[2].Children...)
					}
					return params, nil
				}),
			Epsilon(nil),
		)(i)
	}

	Epsilon = func(n *Node) Consumer {
		return func(i int) (int, *Node, error) {
			return i, n, nil
		}
	}

	Concat = func(consumers ...Consumer) func(func(...*Node)(*Node, error)) Consumer {
		return func(action func(...*Node)(*Node, error)) Consumer {
			return func(i int) (int, *Node, error) {
				var nodes []*Node
				var n *Node
				var err error
				j := i
				for _, consumer := range consumers {
					j, n, err = consumer(j)
					if err == nil {
						nodes = append(nodes, n)
					} else {
						return i, nil, err
					}
				}
				an, aerr := action(nodes...)
				if aerr != nil {
					return i, nil, aerr
				}
				return j, an, nil
			}
		}
	}

	Alt = func(a, b Consumer) Consumer {
		return func(i int) (int, *Node, error) {
			j, na, err_a := a(i)
			k, nb, err_b := b(i)
			if err_a == nil && err_b == nil {
				if j >= k {
					return j, na, nil
				}
				return k, nb, nil
			} else if err_a == nil {
				return j, na, nil
			} else if err_b == nil {
				return k, nb, nil
			}
			return i, nil, fmt.Errorf("%v, %v", err_a, err_b)
		}
	}

	Consume = func(token string) Consumer {
		return func(i int) (int, *Node, error) {
			if i == len(tokens) {
				return i, nil, fmt.Errorf("Ran off the end of the input. Expected %v", token)
			}
			tk := tokens[i]
			if tk.Type == TokMap[token] {
				return i+1, NewTokenNode(tk), nil
			}
			return i, nil, fmt.Errorf("Expected %v got %v", token, tk)
		}
	}
	
	i, node, err := Stmts(0)
	if err != nil {
		return nil, err
	}

	if len(tokens) != i {
		return nil, fmt.Errorf("Unconsumed input, %v", tokens[i:len(tokens)])
	}
	return node, nil
}

