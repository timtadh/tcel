# TCEL - A Language

By Tim Henderson (tadh@case.edu)

This is a simple language with a static type checker. It is an interpreted
expression oriented functional language.  Which features:

- First class functions
- Closures
- Integers, floats, strings, booleans
- Conditional expressions

This language is evolving fast and may have undocumented features or bugs. It
started out as an example I wrote for the compilers class I teach, EECS 337
Compiler Design, at Case Western Reserve University.

#### Running:

    $ go get github.com/timtadh/tcel
    $ tcel ./ex/fib.x

#### Example Computing the Fibonacci Sequence

**file**  `./ex/fib.x`

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
$ tcel ex/fib.x 
> lexing ex/fib.x
> parsing ex/fib.x
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

