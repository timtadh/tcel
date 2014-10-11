package frontend

import (
	"fmt"
	"strconv"
	"strings"
)

import (
	lex "github.com/timtadh/lexmachine"
	"github.com/timtadh/lexmachine/machines"
)

type Token struct {
	lex.Token
	Filename string
}

func NewToken(token int, value interface{}, match *machines.Match, filename string) *Token {
	return &Token{
		Token: lex.Token{
			Type: token,
			Value: value,
			Lexeme: match.Bytes,
			TC: match.TC,
			StartLine: match.StartLine,
			StartColumn: match.StartColumn,
			EndLine: match.EndLine,
			EndColumn: match.EndColumn,
		},
		Filename: filename,
	}
}

func (self *Token) String() string {
	return fmt.Sprintf("'%v' <%v %v>", Tokens[self.Token.Type], self.Token.String(), self.Filename)
}

var Literals []string
var Tokens []string
var TokMap map[string]int

func init() {
	Literals = []string{
		"=",
		"{",
		"}",
		"(",
		")",
		"[",
		"]",
		"+",
		"-",
		"*",
		"/",
		"%",
		",",
		"&&",
		"||",
		"!",
		"<",
		"<=",
		"==",
		"!=",
		">=",
		">",
	}
	Tokens = []string{
		"NAME",
		"FN",
		"IF",
		"ELSE",
		"TRUE",
		"FALSE",
		"INT",
		"FLOAT",
		"STRING",
	}
	Tokens = append(Tokens, Literals...)
	TokMap = make(map[string]int)
	for i, tok := range Tokens {
		TokMap[tok] = i
	}
}

type LexerContext struct {
	Filename string
}

func NewContext(filename string) *LexerContext {
	return &LexerContext{Filename: filename}
}

func (self *LexerContext) Token(name string) lex.Action {
	return func(scan *lex.Scanner, match *machines.Match) (interface{}, error) {
		return NewToken(TokMap[name], string(match.Bytes), match, self.Filename), nil
	}
}

func (self *LexerContext) Skip(scan *lex.Scanner, match *machines.Match) (interface{}, error) {
	return nil, nil
}

func (self *LexerContext) Literal(scan *lex.Scanner, match *machines.Match) (interface{}, error) {
	return NewToken(TokMap[string(match.Bytes)], string(match.Bytes), match, self.Filename), nil
}

func Lexer(text, filename string) (*lex.Scanner, error) {
	ctx := NewContext(filename)
	lexer := lex.NewLexer()

	for _, lit := range Literals {
		r := "\\" + strings.Join(strings.Split(lit, ""), "\\")
		lexer.Add([]byte(r), ctx.Literal)
	}

	lexer.Add([]byte("fn"), ctx.Token("FN"))
	lexer.Add([]byte("if"), ctx.Token("IF"))
	lexer.Add([]byte("else"), ctx.Token("ELSE"))
	lexer.Add([]byte("true"), ctx.Token("TRUE"))
	lexer.Add([]byte("false"), ctx.Token("FALSE"))

	lexer.Add([]byte("([a-z]|[A-Z])([a-z]|[A-Z]|[0-9]|_)*"), ctx.Token("NAME"))
	lexer.Add(
		[]byte("[0-9]+"),
		func(scan *lex.Scanner, match *machines.Match)(interface{}, error) {
			i, err := strconv.Atoi(string(match.Bytes))
			if err != nil {
				return nil, err
			}
			return NewToken(TokMap["INT"], int64(i), match, ctx.Filename), nil
		},
	)
	lexer.Add(
		[]byte("[0-9]*\\.?[0-9]+((E|e)(\\+|-)?[0-9]+)?"),
		func(scan *lex.Scanner, match *machines.Match)(interface{}, error) {
			f, err := strconv.ParseFloat(string(match.Bytes), 64)
			if err != nil {
				return nil, err
			}
			return NewToken(TokMap["FLOAT"], float64(f), match, ctx.Filename), nil
		},
	)
	lexer.Add(
		[]byte("\""),
		func(scan *lex.Scanner, match *machines.Match) (interface{}, error) {
			str := make([]byte, 0, 10)
			match.EndLine = match.StartLine
			match.EndColumn = match.StartColumn
			for tc := scan.TC; tc < len(scan.Text); tc++ {
				match.EndColumn += 1
				if scan.Text[tc] == '\\' {
					// the next character is a literal
					tc++
					match.EndColumn += 1
					if tc < len(scan.Text) {
						switch scan.Text[tc] {
						case 'n', 't', '"': str = append(str, '\\')
						}
					}
				} else if scan.Text[tc] == '"' {
					scan.TC = tc + 1
					return NewToken(TokMap["STRING"], string(str), match, ctx.Filename), nil
				}
				if scan.Text[tc] == '\n' {
					match.EndLine += 1
				}
				str = append(str, scan.Text[tc])
			}
			return nil,
				fmt.Errorf("unclosed string starting at %d, (%d, %d)",
					match.TC, match.StartLine, match.StartColumn)
		},
	)

	lexer.Add([]byte("( |\t|\n)"), ctx.Skip)
	lexer.Add([]byte("//[^\n]*\n"), ctx.Skip)
	lexer.Add([]byte("/\\*"),
		func(scan *lex.Scanner, match *machines.Match)(interface{}, error) {
			for tc := scan.TC; tc < len(scan.Text); tc++ {
				if scan.Text[tc] == '\\' {
					// the next character is skipped
					tc++
				} else if scan.Text[tc] == '*' && tc+1 < len(scan.Text) {
					if scan.Text[tc+1] == '/' {
						scan.TC = tc+2
						return nil, nil
					}
				}
			}
			return nil,
			fmt.Errorf("unclosed comment starting at %d, (%d, %d)",
			match.TC, match.StartLine, match.StartColumn)
		},
	)

	return lexer.Scanner([]byte(text))
}
