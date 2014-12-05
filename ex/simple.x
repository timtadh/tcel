add = fn(x int, y int) int {
	fn(z int) int {
		x + z
	}(y)
}
sub = fn(x int, y int) int {
	fn(a int, b int) int {
		a + x - y + b
	}(-4, 4)
}
print("this program computes (j - (i + ?)) if i is odd")
i = read_stdin_int("enter i =")
j = read_stdin_int("enter j =")
print_int(
	if i % 2 == 0 {
		i
	} else {
		sub(j, add(i, read_stdin_int("add how much to i ?")))
	})
