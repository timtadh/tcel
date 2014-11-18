s = if 1 == 2 {
	"type a number: "
} else {
	"type a Number: "
}
i = read_stdin_int(s)
add = fn(x int, y int) int {
	echo = fn(i int) int { i }
	fn(i int) int { i*2 }(echo(x) + echo(y))
}
print_int(i)
print_int(if i % 2 == 0 {
             add(1, 2)
          } else {
             add(2, 2)
          })
