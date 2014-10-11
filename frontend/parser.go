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

Applies -> Apply Applies'
         | Index Applies

Applies' -> Apply Applies'
          | Index Applies'
          | e

Apply -> ( Params )

Index -> [ Expr ]

Params -> Expr Params'
        | e

Params' -> , Expr Params'
         | e

Factor -> NAME
        | INT
        | FLOAT
        | STRING
        | Function
        | If
        | New
        | ( Expr )

If -> IF BooleanExpr { Stmts } ELSE IfElse

IfElse -> { Stmts }
        | If

Function -> FN ( ParamDecls ) Type { Stmts }

ParamDecls -> NAME Type ParamDecls'
            | e

ParamDecls' -> , NAME Type ParamDecls'
             | e

Type -> NAME
      | FN ( TypeParams ) Type
      | [Expr]Type

TypeParams -> Type TypeParams'
            | e

TypeParams' -> , Type TypeParms'
             | e

BooleanExpr : AndExpr BooleanExpr*
            ;

BooleanExpr* : "|" "|" AndExpr BooleanExpr*
             | e
             ;

AndExpr : NotExpr AndExpr*
        ;

AndExpr* : "&" "&" NotExpr AndExpr*
         | e
         ;

NotExpr : "!" BooleanTerm
        | BooleanTerm
        ;

BooleanTerm : CmpExpr
            | BooleanConstant
            | "(" BooleanExpr ")"
            ;

CmpExpr : Expr CmpOp Expr ;

CmpOp : "<"
      | "<" "="
      | "=" "="
      | "!" "="
      | ">" "="
      | ">"
      ;

BooleanConstant : true
                | false
                ;

New : NEW Type ;
*/

type ParseError struct {
	ErrorFmt string
	Token *Token
}

func Error(fmt string, t *Token) *ParseError {
	return &ParseError{ErrorFmt: fmt, Token: t}
}

func (self *ParseError) Less(o *ParseError) bool {
	if self == nil || o == nil {
		return false
	}
	if self.Token == nil || o.Token == nil {
		return false
	}
	if self.Token.StartLine < o.Token.StartLine {
		return true
	} else if self.Token.StartLine > o.Token.StartLine {
		return false
	}
	if self.Token.StartColumn < o.Token.StartColumn {
		return true
	} else if self.Token.StartColumn > o.Token.StartColumn {
		return false
	}
	if self.Token.EndLine > o.Token.EndLine {
		return true
	} else if self.Token.EndLine < o.Token.EndLine {
		return false
	}
	if self.Token.EndColumn > o.Token.EndColumn {
		return true
	} else if self.Token.EndColumn < o.Token.EndColumn {
		return false
	}
	return false
}

func (self *ParseError) Error() string {
	return fmt.Sprintf(self.ErrorFmt, self.Token)
}

type Consumer interface {
	Consume(i int) (int, *Node, *ParseError)
}

type StringConsumer struct {
	name string
	Productions map[string]Consumer
}

func (self StringConsumer) Consume(i int) (int, *Node, *ParseError) {
	return self.Productions[self.name].Consume(i)
}

type FnConsumer func(i int) (int, *Node, *ParseError)

func (self FnConsumer) Consume(i int) (int, *Node, *ParseError) {
	return self(i)
}


func Parse(tokens []*Token) (*Node, error) {

	type Result struct {
		j int
		n *Node
		e *ParseError
	}

	P := make(map[string]Consumer)

	SC := func(name string) StringConsumer {
		return StringConsumer{
			name: name,
			Productions: P,
		}
	}

	var (
		/*
		Stmts, Stmt, Assign, Expr, Expr_, Term, Term_, Unary, PostUnary, Factor,
		Applies, Applies_, Params, Params_, Apply, Index, Function, ParamDecls,
		ParamDecls_, Type, TypeParams, TypeParams_, If, IfElse, BooleanExpr,
		BooleanExpr_, AndExpr, AndExpr_, NotExpr, BooleanTerm, CmpExpr,
		BooleanConstant, CmpOp, Array, ArrayLiteral, ArrayParams, ArrayParams_ Consumer */
		Epsilon func(*Node) Consumer
		Consume func(string) Consumer
		Concat func(...Consumer) func(func(...*Node)(*Node, *ParseError)) Consumer
		Alt func(...Consumer) Consumer
	)

	var top_err *ParseError = nil

	collapse := func(subtree, extra *Node) *Node {
		if extra == nil {
			return subtree
		}
		return extra.Get(0).AddKid(subtree).AddKid(extra.Get(1))
	}

	swing := func (nodes ...*Node) (*Node, *ParseError) {
		n := NewNode("T").AddKid(nodes[0]).AddKid(collapse(nodes[1], nodes[2]))
		return n, nil
	}

	Epsilon = func(n *Node) Consumer {
		return FnConsumer(func(i int) (int, *Node, *ParseError) {
			return i, n, nil
		})
	}

	Cached := func(c Consumer) Consumer {
		cache := make(map[int]*Result)
		return FnConsumer(func(i int) (int, *Node, *ParseError) {
			if res, in := cache[i]; in {
				return res.j, res.n, res.e
			}
			j, n, e := c.Consume(i)
			cache[i] = &Result{j, n, e}
			return j, n, e
		})
	}

	Concat = func(consumers ...Consumer) func(func(...*Node)(*Node, *ParseError)) Consumer {
		return func(action func(...*Node)(*Node, *ParseError)) Consumer {
			return Cached(FnConsumer(func(i int) (int, *Node, *ParseError) {
				var nodes []*Node
				var n *Node
				var err *ParseError
				j := i
				for _, c := range consumers {
					j, n, err = c.Consume(j)
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
			}))
		}
	}

	Alt = func(consumers ...Consumer) Consumer {
		return Cached(FnConsumer(func(i int) (int, *Node, *ParseError) {
			var err *ParseError = nil
			for _, c := range consumers {
				j, n, e := c.Consume(i)
				if e == nil {
					return j, n, nil
				} else if err == nil || err.Less(e) {
					err = e
				}
			}
			if top_err == nil || top_err.Less(err) {
				top_err = err
			}
			return i, nil, err
		}))
	}

	Consume = func(token string) Consumer {
		return FnConsumer(func(i int) (int, *Node, *ParseError) {
			if i == len(tokens) {
				return i, nil, Error(
					fmt.Sprintf("Ran off the end of the input. Expected %v. %%v", token), nil)
			}
			tk := tokens[i]
			if tk.Type == TokMap[token] {
				return i+1, NewTokenNode(tk), nil
			}
			return i, nil, Error(fmt.Sprintf("Expected %v got %%v", token), tk)
		})
	}

	for _, token := range Tokens {
		P[token] = Consume(token)
	}

	P["Stmts"] = Alt(
		Concat(SC("Stmt"), SC("Stmts"))(func (nodes ...*Node) (*Node, *ParseError) {
			stmts := NewNode("Stmts").AddKid(nodes[0])
			stmts.Children = append(stmts.Children, nodes[1].Children...)
			return stmts, nil
		}),
		Concat(SC("Stmt"))(func (nodes ...*Node) (*Node, *ParseError) {
			stmts := NewNode("Stmts").AddKid(nodes[0])
			return stmts, nil
		}),
	)

	P["Stmt"] = Alt(SC("Assign"), SC("Expr"), SC("BooleanTerm"))

	P["Assign"] = Concat(SC("NAME"), SC("="), SC("Expr"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			stmts := NewNode("Assign").AddKid(nodes[0]).AddKid(nodes[2])
			return stmts, nil
		})

	P["Expr"] = Concat(SC("Term"), SC("Expr_"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return collapse(nodes[0], nodes[1]), nil
		})

	P["Expr_"] = Alt(
		Concat(SC("+"), SC("Term"), SC("Expr_"))(swing),
		Concat(SC("-"), SC("Term"), SC("Expr_"))(swing),
		Epsilon(nil),
	)

	P["Term"] = Concat(SC("Unary"), SC("Term_"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return collapse(nodes[0], nodes[1]), nil
		})

	P["Term_"] = Alt(
		Concat(SC("*"), SC("Unary"), SC("Term_"))(swing),
		Concat(SC("/"), SC("Unary"), SC("Term_"))(swing),
		Concat(SC("%"), SC("Unary"), SC("Term_"))(swing),
		Epsilon(nil),
	)

	P["Unary"] = Alt(
		SC("PostUnary"),
		Concat(SC("-"), SC("PostUnary"))(func (nodes ...*Node) (*Node, *ParseError) {
			nodes[0].Label = "Negate"
			return nodes[0].AddKid(nodes[1]), nil
		}),
	)

	P["PostUnary"] = Alt(
		Concat(SC("Factor"), SC("Applies"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1].AddLeftMostKid(nodes[0], "Call", "Index"), nil
			}),
		SC("Factor"),
	)

	aply := func(name string) func(...*Node) (*Node, *ParseError) {
		return func(nodes ...*Node) (*Node, *ParseError) {
			if nodes[1] == nil {
				return NewNode(name).AddKid(nodes[0]), nil
			}
			root := nodes[1]
			root.AddLeftMostKid(NewNode(name).AddKid(nodes[0]), "Call", "Index")
			return root, nil
		}
	}

	P["Applies"] = Alt(
		Concat(SC("Apply"), SC("Applies_"))(aply("Call")),
		Concat(SC("Index"), SC("Applies_"))(aply("Index")),
	)

	P["Applies_"] = Alt(
		Concat(SC("Apply"), SC("Applies_"))(aply("Call")),
		Concat(SC("Index"), SC("Applies_"))(aply("Index")),
		Epsilon(nil),
	)

	P["Apply"] = Concat(SC("("), SC("Params"), SC(")"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return nodes[1], nil
		})

	P["Index"] = Concat(SC("["), SC("Expr"), SC("]"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return nodes[1], nil
		})

	P["Params"] = Alt(
		Concat(SC("Expr"), SC("Params_"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				params := NewNode("Params").AddKid(nodes[0])
				if nodes[1] != nil {
					params.Children = append(params.Children, nodes[1].Children...)
				}
				return params, nil
			}),
		Epsilon(NewNode("Params")),
	)

	P["Params_"] = Alt(
		Concat(SC(","), SC("Expr"), SC("Params_"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				params := NewNode("Params").AddKid(nodes[1])
				if nodes[2] != nil {
					params.Children = append(params.Children, nodes[2].Children...)
				}
				return params, nil
			}),
		Epsilon(nil),
	)

	P["Factor"] = Alt(
		SC("NAME"),
		SC("INT"),
		SC("FLOAT"),
		SC("STRING"),
		SC("Function"),
		SC("If"),
		SC("New"),
		Concat(SC("("), SC("Expr"), SC(")"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1], nil
			}),
	)


	P["New"] = Concat(
		SC("NEW"), SC("Type"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return nodes[0].AddKid(nodes[1]), nil
		})

	P["Function"] = Concat(
		SC("FN"), SC("("), SC("ParamDecls"), SC(")"),
		SC("Type"), SC("{"), SC("Stmts"), SC("}"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			n := NewNode("Func").AddKid(nodes[2]).AddKid(nodes[4]).AddKid(nodes[6])
			return n, nil
		})

	P["ParamDecls"] = Alt(
		Concat(SC("NAME"), SC("Type"), SC("ParamDecls_"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				params := NewNode("ParamDecls").AddKid(
					NewNode("ParamDecl").AddKid(nodes[0]).AddKid(nodes[1]))
				if nodes[2] != nil {
					params.Children = append(params.Children, nodes[2].Children...)
				}
				return params, nil
			}),
		Epsilon(NewNode("ParamDecls")),
	)

	P["ParamDecls_"] = Alt(
		Concat(SC(","), SC("NAME"), SC("Type"), SC("ParamDecls_"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				params := NewNode("ParamDecls").AddKid(
					NewNode("ParamDecl").AddKid(nodes[1]).AddKid(nodes[2]))
				if nodes[3] != nil {
					params.Children = append(params.Children, nodes[3].Children...)
				}
				return params, nil
			}),
		Epsilon(nil),
	)

	P["Type"] = Alt(
		Concat(SC("NAME"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return NewNode("TypeName").AddKid(nodes[0]), nil
			}),
		Concat(SC("FN"), SC("("), SC("TypeParams"), SC(")"), SC("Type"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				n := NewNode("FuncType").AddKid(nodes[2]).AddKid(nodes[4])
				return n, nil
			}),
		Concat(SC("["), SC("Expr"), SC("]"), SC("Type"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				n := NewNode("ArrayType").AddKid(nodes[3]).AddKid(nodes[1])
				return n, nil
			}),
	)

	P["TypeParams"] = Alt(
		Concat(SC("Type"), SC("TypeParams_"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				params := NewNode("TypeParams").AddKid(nodes[0])
				if nodes[1] != nil {
					params.Children = append(params.Children, nodes[1].Children...)
				}
				return params, nil
			}),
		Epsilon(NewNode("TypeParams")),
	)

	P["TypeParams_"] = Alt(
		Concat(SC(","), SC("Type"), SC("TypeParams_"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				params := NewNode("TypeParams").AddKid(nodes[1])
				if nodes[2] != nil {
					params.Children = append(params.Children, nodes[2].Children...)
				}
				return params, nil
			}),
		Epsilon(nil),
	)

	P["If"] = Concat(
		SC("IF"), SC("BooleanExpr"), SC("{"), SC("Stmts"), SC("}"),
		SC("ELSE"), SC("IfElse"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				n := NewNode("If").AddKid(nodes[1]).AddKid(nodes[3]).AddKid(nodes[6])
				return n, nil
			})

	P["IfElse"] = Alt(
		Concat(SC("If"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return NewNode("Stmts").AddKid(nodes[0]), nil
			}),
		Concat(SC("{"), SC("Stmts"), SC("}")) (
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1], nil
			}),
	)

	P["BooleanExpr"] = Concat(SC("AndExpr"), SC("BooleanExpr_"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return collapse(nodes[0], nodes[1]), nil
		})

	P["BooleanExpr_"] = Alt(
		Concat(SC("||"), SC("AndExpr"), SC("BooleanExpr_"))(swing),
		Epsilon(nil),
	)

	P["AndExpr"] = Concat(SC("NotExpr"), SC("AndExpr_"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return collapse(nodes[0], nodes[1]), nil
		})

	P["AndExpr_"] = Alt(
		Concat(SC("&&"), SC("NotExpr"), SC("AndExpr_"))(swing),
		Epsilon(nil),
	)

	P["NotExpr"] = Alt(
		Concat(SC("!"), SC("BooleanTerm"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return NewNode("!").AddKid(nodes[1]), nil
			}),
		SC("BooleanTerm"),
	)

	P["BooleanTerm"] = Alt(
		Alt(SC("CmpExpr"), SC("BooleanConstant")),
		Concat(SC("("), SC("BooleanExpr"), SC(")"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1], nil
			}),
	)

	P["CmpExpr"] = Concat(SC("Expr"), SC("CmpOp"), SC("Expr"))(
		func (nodes ...*Node) (*Node, *ParseError) {
			return nodes[1].AddKid(nodes[0]).AddKid(nodes[2]), nil
		})

	P["CmpOp"] = Alt(SC("<"), SC("<="), SC("=="), SC("!="), SC(">"), SC(">="))

	P["BooleanConstant"] = Alt(SC("TRUE"), SC("FALSE"))

	i, node, err := P["Stmts"].Consume(0)

	if err != nil {
		return nil, err
	}

	if len(tokens) != i {
		return nil, top_err
	}
	return node, nil
}
