package main

import (
	"os"
	"io"
	"fmt"
	"io/ioutil"
	logpkg "log"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/cwru-compilers/type-check-example/frontend"
	"github.com/cwru-compilers/type-check-example/checker"
	"github.com/cwru-compilers/type-check-example/evaluator"
)

var log *logpkg.Logger

func init() {
	log = logpkg.New(os.Stderr, "", 0)
}


var UsageMessage string = "type-check-example -o <path> <input>+ "
var ExtendedMessage string = `

Options
    -h, --help                          print this message
    -o, output=<path>                   output path
    -L, lex                             stop at lexing
    -A, ast                             stop at AST generation
    -T, typed-ast                       stop at type checked AST

Specs
    <path>
        A file system path to an existing file
`

func Usage(code int) {
    fmt.Fprintln(os.Stderr, UsageMessage)
    if code == 0 {
        fmt.Fprintln(os.Stderr, ExtendedMessage)
        code = 1
    } else {
        fmt.Fprintln(os.Stderr, "Try -h or --help for help")
    }
    os.Exit(code)
}

func write(X interface{String() string}, f io.Writer) {
	f.Write([]byte(X.String()))
	f.Write([]byte("\n"))
	return
}

type FileTokens struct {
	Filename string
	Tokens []*frontend.Token
}

func (self *FileTokens) String() string {
	return fmt.Sprintf("%v %v", self.Filename, self.Tokens)
}

type FilesTokens []*FileTokens

func (self FilesTokens) String() string {
	return fmt.Sprintf("%v", []*FileTokens(self))
}

func lex(paths ... string) FilesTokens {
	defer func() {
		if e := recover(); e != nil {
			log.Fatal(e)
		}
	}()
	var files []*FileTokens
	for _, path := range paths {
		log.Print("> lexing ", path)
		f, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		program, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}
		scanner, err := frontend.Lexer(string(program), path)
		if err != nil {
			log.Fatal(err)
		}
		var tokens []*frontend.Token
		for tok, err, eof := scanner.Next(); !eof; tok, err, eof = scanner.Next() {
			if err != nil {
				panic(err)
			}
			tokens = append(tokens, tok.(*frontend.Token))
		}
		files = append(files, &FileTokens{path, tokens})
	}
	return files
}

func parse(files FilesTokens) *frontend.Node {
	defer func() {
		if e := recover(); e != nil {
			log.Fatal(e)
		}
	}()
	var A *frontend.Node = nil
	for _, file := range files {
		log.Println("> parsing", file.Filename)
		n, err := frontend.Parse(file.Tokens)
		if err != nil {
			log.Fatal(err)
		}

		if A == nil {
			A = n
		} else {
			for _, kid := range n.Children {
				A.AddKid(kid)
			}
		}
	}
	if A == nil {
		log.Fatal("You must supply input paths")
	}
	return A
}

func typecheck(node *frontend.Node) *frontend.Node {
	log.Print("> type checking")
	err := checker.Check(node)
	if err != nil {
		log.Fatal(err)
	}
	return node
}

func eval(node *frontend.Node) []interface{} {
	log.Print("> evaluating")
	values, err := evaluator.Evaluate(node)
	if err != nil {
		log.Fatal(err)
	}
	return values
}

func main() {

	short := "ho:LAT"
	long := []string{
		"help",
		"output=",
		"lex", "ast", "typed-ast",
	}

	args, optargs, err := getopt.GetOpt(os.Args[1:], short, long)
	if err != nil {
		log.Print(os.Stderr, err)
		Usage(1)
	}


	output := ""
	stop_at := "run"
	for _, oa := range optargs {
		switch oa.Opt() {
		case "-h", "--help": Usage(0)
		case "-o", "--output":
			output = oa.Arg()
		case "-L", "--lex":
			stop_at = "lex"
		case "-A", "--ast":
			stop_at = "ast"
		case "-T", "--typed-ast":
			stop_at = "typed-ast"
		}
	}

	var ouf io.Writer
	if output == "" {
		ouf = os.Stdout
	} else {
		f, err := os.Create(output)
		if err != nil {
			log.Print(os.Stderr, err)
			Usage(1)
		}
		defer f.Close()
		ouf = f
	}

	if len(args) <= 0 {
		log.Print("Must supply some input paths")
		Usage(1)
	}

	L := lex(args...)
	if stop_at == "lex" {
		write(L, ouf)
		return
	}

	A := parse(L)
	if stop_at == "ast" {
		ouf.Write([]byte(fmt.Sprintf("%v\n", A.Serialize(true))))
		return
	}

	T := typecheck(A)
	if stop_at == "typed-ast" {
		ouf.Write([]byte(fmt.Sprintf("%v\n", T.Serialize(false))))
		return
	}

	values := eval(T)
	for _, value := range values {
		ouf.Write([]byte(fmt.Sprintf("%v\n", value)))
	}
}

