fn(x int) fn(int) fn(int) int {
	fn(y int) fn(int) int {
		fn(z int) int {
			x + y + z
		}
	}
}(1)(2)(3)
