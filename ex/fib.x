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
