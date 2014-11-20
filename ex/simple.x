s = if 2 == 2 {
	"type a number: "
} else {
	"Type a Number: "
}
i = read_stdin_int(s)
add = fn(x int, y int) int {
	fn(z int) int {
		x + z
	}(y)
}
print_int(i)
print_int(if i % 2 == 0 {
             i
          } else {
             add(i, 5)
          })

