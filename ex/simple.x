add = fn(x int, y int) int { x + y }
print_int(if 1 == 2 {
             add(1, 2)
          } else {
             add(2, 2)
          })


