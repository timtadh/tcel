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
        | Array
        | Function
        | If
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

Array -> ArrayLiteral
       | [ Expr ] Type

ArrayLiteral -> [ ArrayParams ]

ArrayParams -> Expr Params'

ArrayParams' -> , Expr Params'
              | e
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

func Parse(tokens []*Token) (*Node, error) {

	type Consumer func(i int) (int, *Node, *ParseError)
	var (
		Stmts, Stmt, Assign, Expr, Expr_, Term, Term_, Unary, PostUnary, Factor,
		Applies, Applies_, Params, Params_, Apply, Index, Function, ParamDecls,
		ParamDecls_, Type, TypeParams, TypeParams_, If, IfElse, BooleanExpr,
		BooleanExpr_, AndExpr, AndExpr_, NotExpr, BooleanTerm, CmpExpr,
		BooleanConstant, CmpOp/*, Array, ArrayLiteral, ArrayParams, ArrayParams_*/ Consumer
		Epsilon func(*Node) Consumer
		Consume func(string) Consumer
		Concat func(...Consumer) func(func(...*Node)(*Node, *ParseError)) Consumer
		Alt func(... Consumer) Consumer
	)

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

	Stmts = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Stmt, Stmts)(func (nodes ...*Node) (*Node, *ParseError) {
				stmts := NewNode("Stmts").AddKid(nodes[0])
				stmts.Children = append(stmts.Children, nodes[1].Children...)
				return stmts, nil
			}),
			Concat(Stmt)(func (nodes ...*Node) (*Node, *ParseError) {
				stmts := NewNode("Stmts").AddKid(nodes[0])
				return stmts, nil
			}),
		)(i)
	}

	Stmt = func(i int) (int, *Node, *ParseError) {
		return Alt(Assign, Expr)(i)
	}

	Assign = func(i int) (int, *Node, *ParseError) {
		return Concat(Consume("NAME"), Consume("="), Expr)(
			func (nodes ...*Node) (*Node, *ParseError) {
				stmts := NewNode("Assign").AddKid(nodes[0]).AddKid(nodes[2])
				return stmts, nil
			})(i)
	}

	Expr = func(i int) (int, *Node, *ParseError) {
		return Concat(Term, Expr_)(
			func (nodes ...*Node) (*Node, *ParseError) {
				return collapse(nodes[0], nodes[1]), nil
			})(i)
	}

	Expr_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("+"), Term, Expr_)(swing),
			Concat(Consume("-"), Term, Expr_)(swing),
			Epsilon(nil),
		)(i)
	}

	Term = func(i int) (int, *Node, *ParseError) {
		return Concat(Unary, Term_)(
			func (nodes ...*Node) (*Node, *ParseError) {
				return collapse(nodes[0], nodes[1]), nil
			})(i)
	}

	Term_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("*"), Unary, Term_)(swing),
			Concat(Consume("/"), Unary, Term_)(swing),
			Concat(Consume("%"), Unary, Term_)(swing),
			Epsilon(nil),
		)(i)
	}

	Unary = func(i int) (int, *Node, *ParseError) {
		return Alt(
			PostUnary,
			Concat(Consume("-"), PostUnary)(func (nodes ...*Node) (*Node, *ParseError) {
				nodes[0].Label = "Negate"
				return nodes[0].AddKid(nodes[1]), nil
			}),
		)(i)
	}

	PostUnary = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Factor, Applies)(
				func (nodes ...*Node) (*Node, *ParseError) {
					return nodes[1].AddLeftMostKid(nodes[0], "Call"), nil
				}),
			Factor,
		)(i)
	}

	aply := func(name string) func(...*Node) (*Node, *ParseError) {
		return func(nodes ...*Node) (*Node, *ParseError) {
			if nodes[1] == nil {
				return NewNode(name).AddKid(nodes[0]), nil
			}
			root := nodes[1]
			root.AddLeftMostKid(NewNode(name).AddKid(nodes[0]), name)
			return root, nil
		}
	}

	Applies = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Apply, Applies_)(aply("Call")),
			Concat(Index, Applies_)(aply("Index")),
		)(i)
	}

	Applies_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Apply, Applies_)(aply("Call")),
			Concat(Index, Applies_)(aply("Index")),
			Epsilon(nil),
		)(i)
	}

	Apply = func(i int) (int, *Node, *ParseError) {
		return Concat(Consume("("), Params, Consume(")"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1], nil
			})(i)
	}

	Index = func(i int) (int, *Node, *ParseError) {
		return Concat(Consume("["), Params, Consume("]"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1], nil
			})(i)
	}

	Params = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Expr, Params_)(
				func (nodes ...*Node) (*Node, *ParseError) {
					params := NewNode("Params").AddKid(nodes[0])
					if nodes[1] != nil {
						params.Children = append(params.Children, nodes[1].Children...)
					}
					return params, nil
				}),
			Epsilon(NewNode("Params")),
		)(i)
	}

	Params_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume(","), Expr, Params_)(
				func (nodes ...*Node) (*Node, *ParseError) {
					params := NewNode("Params").AddKid(nodes[1])
					if nodes[2] != nil {
						params.Children = append(params.Children, nodes[2].Children...)
					}
					return params, nil
				}),
			Epsilon(nil),
		)(i)
	}

	Factor = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Consume("NAME"),
			Consume("INT"),
			Consume("FLOAT"),
			Consume("STRING"),
			Function,
			If,
			Concat(Consume("("), Expr, Consume(")"))(
				func (nodes ...*Node) (*Node, *ParseError) {
					return nodes[1], nil
				}),
		)(i)
	}

	Function = func(i int) (int, *Node, *ParseError) {
		return Concat(
			Consume("FN"), Consume("("), ParamDecls, Consume(")"), Type,
			Consume("{"), Stmts, Consume("}"))(
			func (nodes ...*Node) (*Node, *ParseError) {
				n := NewNode("Func").AddKid(nodes[2]).AddKid(nodes[4]).AddKid(nodes[6])
				return n, nil
			})(i)
	}

	ParamDecls = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("NAME"), Type, ParamDecls_)(
				func (nodes ...*Node) (*Node, *ParseError) {
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

	ParamDecls_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume(","), Consume("NAME"), Type, ParamDecls_)(
				func (nodes ...*Node) (*Node, *ParseError) {
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

	Type = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("NAME"))(
				func (nodes ...*Node) (*Node, *ParseError) {
					return NewNode("TypeName").AddKid(nodes[0]), nil
				}),
			Concat(Consume("FN"), Consume("("), TypeParams, Consume(")"), Type)(
				func (nodes ...*Node) (*Node, *ParseError) {
					n := NewNode("FuncType").AddKid(nodes[2]).AddKid(nodes[4])
					return n, nil
				}),
		)(i)
	}

	TypeParams = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Type, TypeParams_)(
				func (nodes ...*Node) (*Node, *ParseError) {
					params := NewNode("TypeParams").AddKid(nodes[0])
					if nodes[1] != nil {
						params.Children = append(params.Children, nodes[1].Children...)
					}
					return params, nil
				}),
			Epsilon(NewNode("TypeParams")),
		)(i)
	}

	TypeParams_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume(","), Type, TypeParams_)(
				func (nodes ...*Node) (*Node, *ParseError) {
					params := NewNode("TypeParams").AddKid(nodes[1])
					if nodes[2] != nil {
						params.Children = append(params.Children, nodes[2].Children...)
					}
					return params, nil
				}),
			Epsilon(nil),
		)(i)
	}

	If = func(i int) (int, *Node, *ParseError) {
		return Concat(
			Consume("IF"), BooleanExpr, Consume("{"), Stmts, Consume("}"),
			Consume("ELSE"), IfElse)(
				func (nodes ...*Node) (*Node, *ParseError) {
					n := NewNode("If").AddKid(nodes[1]).AddKid(nodes[3]).AddKid(nodes[6])
					return n, nil
				})(i)
	}

	IfElse = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(If)(
				func (nodes ...*Node) (*Node, *ParseError) {
					return NewNode("Stmts").AddKid(nodes[0]), nil
				}),
			Concat(Consume("{"), Stmts, Consume("}")) (
				func (nodes ...*Node) (*Node, *ParseError) {
					return nodes[1], nil
				}),
		)(i)
	}

	BooleanExpr = func(i int) (int, *Node, *ParseError) {
		return Concat(AndExpr, BooleanExpr_)(
			func (nodes ...*Node) (*Node, *ParseError) {
				return collapse(nodes[0], nodes[1]), nil
			})(i)
	}

	BooleanExpr_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("||"), AndExpr, BooleanExpr_)(swing),
			Epsilon(nil),
		)(i)
	}

	AndExpr = func(i int) (int, *Node, *ParseError) {
		return Concat(NotExpr, AndExpr_)(
			func (nodes ...*Node) (*Node, *ParseError) {
				return collapse(nodes[0], nodes[1]), nil
			})(i)
	}

	AndExpr_ = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("&&"), NotExpr, AndExpr_)(swing),
			Epsilon(nil),
		)(i)
	}

	NotExpr = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Concat(Consume("!"), BooleanTerm)(
				func (nodes ...*Node) (*Node, *ParseError) {
					return NewNode("!").AddKid(nodes[1]), nil
				}),
			BooleanTerm,
		)(i)
	}

	BooleanTerm = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Alt(CmpExpr, BooleanConstant),
			Concat(Consume("("), BooleanExpr, Consume(")"))(
				func (nodes ...*Node) (*Node, *ParseError) {
					return nodes[1], nil
				}),
		)(i)
	}

	CmpExpr = func(i int) (int, *Node, *ParseError) {
		return Concat(Expr, CmpOp, Expr)(
			func (nodes ...*Node) (*Node, *ParseError) {
				return nodes[1].AddKid(nodes[0]).AddKid(nodes[2]), nil
			})(i)
	}

	CmpOp = func(i int) (int, *Node, *ParseError) {
		return Alt(
			Consume("<"), Consume("<="),
			Consume("=="), Consume("!="),
			Consume(">"), Consume(">="),
		)(i)
	}

	BooleanConstant = func(i int) (int, *Node, *ParseError) {
		return Alt(Consume("TRUE"), Consume("FALSE"))(i)
	}

	Epsilon = func(n *Node) Consumer {
		return func(i int) (int, *Node, *ParseError) {
			return i, n, nil
		}
	}

	Concat = func(consumers ...Consumer) func(func(...*Node)(*Node, *ParseError)) Consumer {
		return func(action func(...*Node)(*Node, *ParseError)) Consumer {
			return func(i int) (int, *Node, *ParseError) {
				var nodes []*Node
				var n *Node
				var err *ParseError
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


	/*
	Alt = func(consumers ...Consumer) Consumer {
		return func(i int) (int, *Node, *ParseError) {
			type ret struct {
				j int
				n *Node
				e *ParseError
			}
			var wg sync.WaitGroup
			results := make(chan *ret)
			wg.Add(len(consumers))
			for _, c := range consumers {
				go func(c Consumer) {
					j, n, err := c(i)
					results <- &ret{j, n, err}
					wg.Done()
				}(c)
			}
			go func() {
				wg.Wait()
				close(results)
			}()

			var winner *ret
			var err *ParseError = fmt.Errorf("")
			for res := range results {
				if res.e == nil {
					if winner == nil {
						winner = res
					} else if winner.j < res.j {
						winner = res
					}
				} else {
					err = fmt.Errorf("%v\n%v", res.e, err)
				}
			}

			if winner == nil {
				return i, nil, err
			}
			return winner.j, winner.n, nil
		}
	}*/

	Alt = func(consumers ...Consumer) Consumer {
		return func(i int) (int, *Node, *ParseError) {
			
			type ret struct {
				j int
				n *Node
			}

			var err *ParseError = nil
			for _, c := range consumers {
				j, n, e := c(i)
				if e == nil {
					return j, n, nil
				} else if err == nil || err.Less(e) {
					err = e
				}
			}
			return i, nil, err
		}
	}

	Consume = func(token string) Consumer {
		return func(i int) (int, *Node, *ParseError) {
			if i == len(tokens) {
				return i, nil, Error(
					fmt.Sprintf("Ran off the end of the input. Expected %v. %%v", token), nil)
			}
			tk := tokens[i]
			if tk.Type == TokMap[token] {
				return i+1, NewTokenNode(tk), nil
			}
			return i, nil, Error(fmt.Sprintf("Expected %v got %%v", token), tk)
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

