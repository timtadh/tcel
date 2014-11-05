newint = fn(i int) box(int) {
	a = new int
	^a = i
	a
}
x = newint(5)
y = x
^y
^x = ^x + 1
^y

