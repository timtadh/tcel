f = fn(x int) fn(int) fn(int) int {
	fn(y int) fn(int) int {
		fn(z int) int {
			x + y + z
		}
	}
}
f
g = f(1)
g
h = g(7)
h
i = h(2)
i

