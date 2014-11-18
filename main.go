package main

import (
	"os"
	"io"
	"io/ioutil"
	"fmt"
	"os/exec"
	"syscall"
	"strings"
	"runtime"
	logpkg "log"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/tcel/frontend"
	"github.com/timtadh/tcel/checker"
	"github.com/timtadh/tcel/evaluator"
	"github.com/timtadh/tcel/il"
	"github.com/timtadh/tcel/x86"
)

var log *logpkg.Logger

func init() {
	log = logpkg.New(os.Stderr, "", 0)
	runtime.GOMAXPROCS(4)
}


var UsageMessage string = "tcel -o <path> <input>+ "
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

func call(cmd string) (output string, code int) {
	args := strings.Split(cmd, " ")
	cmd, err := exec.LookPath(args[0])
	if err != nil {
		log.Fatal(err)
	}
	c := exec.Command(cmd, args[1:]...)
	log.Print("> ", cmd, " ", strings.Join(args[1:]," "))
	o, err := c.CombinedOutput()
	if err != nil {
		log.Print(string(o))
		log.Fatal(err)
	}
	if err != nil {
		msg, ok := err.(*exec.ExitError)
		if ok {
			return string(o), msg.Sys().(syscall.WaitStatus).ExitStatus()
		}
		log.Fatal(err)
	}
	return string(o), 0
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
	/*
	defer func() {
		if e := recover(); e != nil {
			log.Fatal(e)
		}
	}()*/
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
	/*
	defer func() {
		if e := recover(); e != nil {
			log.Fatal(e)
		}
	}()*/
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

func ilgen(node *frontend.Node) il.Functions {
	log.Print("> generating intermediate code")
	fns, err := il.Generate(node)
	if err != nil {
		log.Fatal(err)
	}
	return fns
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

func x86_gen(I il.Functions) string {
	log.Print("> compiling intermediate code to x86 32 bit assembly")
	asm, e := x86.Generate(I)
	if e != nil {
		log.Fatal(e)
	}
	return asm
}

func write_lib(lib string) {
	f, err := os.Create(lib)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.Write([]byte(x86.Lib))
}

func link(input, lib, output string) {
	log.Print("> assembling and linking using gcc")
	call("gcc -m32 -c -o lib.o " + lib)
	defer os.Remove("lib.o")
	call("gcc -m32 -c -o main.o " + input)
	defer os.Remove("main.o")
	call("gcc -m32 -o " + output + " lib.o main.o")
}

func main() {

	short := "ho:LATIS"
	long := []string{
		"help",
		"output=",
		"lex", "ast", "typed-ast", "il", "asm", "eval",
	}

	args, optargs, err := getopt.GetOpt(os.Args[1:], short, long)
	if err != nil {
		log.Print(os.Stderr, err)
		Usage(1)
	}


	output := ""
	stop_at := "link"
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
		case "-I", "--il":
			stop_at = "il"
		case "-S", "--asm":
			stop_at = "asm"
		case "--eval":
			stop_at = "eval"
		}
	}

	binary := output
	if stop_at == "link" {
		if output == "" {
			binary = "a.out"
		} else {
			binary = output
		}
		output = "a.s"
	}

	var ouf io.WriteCloser
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
		ouf.Write([]byte(fmt.Sprintf("%v\n", A.Serialize(false))))
		return
	}

	if stop_at == "typed-ast" {
		T := typecheck(A)
		ouf.Write([]byte(fmt.Sprintf("%v\n", T.Serialize(true))))
		return
	}
	
	if stop_at == "eval" {
		values := eval(typecheck(A))
		for _, value := range values {
			ouf.Write([]byte(fmt.Sprintf("%v\n", value)))
		}
	} else {
		I := ilgen(typecheck(A))
		if stop_at == "il" {
			write(I, ouf)
			return
		}

		log.Println(I)
		asm := x86_gen(I)
		ouf.Write([]byte(asm))

		if stop_at == "asm" {
			return
		}


		ouf.Close()
		defer os.Remove(output)

		log.Println(asm)

		lib := "lib.c"
		write_lib(lib)
		defer os.Remove(lib)
		link(output, lib, binary)
	}
}

