add = fn(x int) fn(int) int {
	fn(y int) int {
		x + y
	}
}

fib = fn(add fn(int) fn(int) int) fn(int) int {
	fn(i int) int {
		if i < 0 {
			0
		} else if i <= 1 {
			1
		} else {
			add(self(i-1))(self(i-2))
		}
	}
}

fibm = fn(add fn(int) fn(int) int) fn(int)int {
	fn(n int) int {
		fib = fn(arr []int, i int) []int {
			if i > n {
				arr
			} else {
				if i <= 1 {
					arr[i] = 1
				} else {
					arr[i] = add(arr[i-1])(arr[i-2])
				}
				self(arr, i+1)
			}
		}
		fib(new [n+1]int, 0)[n]
	}
}

use = "array"
n = 25
"using " + use
f = if use == "array" {
		fibm(add)
	} else if use == "recurse" {
		fib(add)
	} else {
		fn(i int) int { -1 }
	}
x = f(n)
x
if x < 0 {
	"fail"
} else {
	"ok"
}

