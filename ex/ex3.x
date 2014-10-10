fib = fn(i int) int {
	if i <= 1 {
		1
	} else {
		self(i-1) + self(i-2)
	}
}

fib(0)
fib(1)
fib(2)
fib(3)
fib(4)
fib(5)
fib(6)
