i = read_stdin_int("type a number: ")
add = fn(x int, y int) int { x + y }
print_int(i)
print_int(if i % 2 == 0 {
             add(1, 2)
          } else {
             add(2, 2)
          })
