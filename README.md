# TCEL - A Language

This is a simple language with a type checker. It is an interpreted language
expression oriented functional language.  Which features:

	- First class functions
	- Closures
	- Integers, floats, strings

#### Running:

    $ go get github.com/cwru-compilers/type-check-example
    $ type-check-example -T ./ex/simple.x

#### Example Computing the Fibonacci Sequence

**file**  `./ex/ex3.x`

```
add = fn(x int) fn(int) int {
	fn(y int) int {
		x + y
	}
}

fib = fn(i int) int {
	if i <= 1 {
		1
	} else {
		add(self(i-1))(self(i-2))
	}
}

fib(0)
fib(1)
fib(2)
fib(3)
fib(4)
fib(5)
fib(6)
```

```
$ ./bin/type-check-example ex/ex3.x 
> lexing ex/ex3.x
> parsing ex/ex3.x
> type checking
> evaluating
unit
unit
1
1
2
3
5
8
13
```

