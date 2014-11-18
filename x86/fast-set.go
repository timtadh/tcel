package x86

import (
	"fmt"
	"strings"
)

type FastSet struct {
	dense []uint
	sparse []uint
}

func NewFastSet(n uint) *FastSet {
	return &FastSet{
		dense: make([]uint, 0, n),
		sparse: make([]uint, n),
	}
}

func (self *FastSet) N() uint {
	return uint(cap(self.dense))
}

func (self *FastSet) Len() int {
	return len(self.dense)
}

func (self *FastSet) Clear() *FastSet {
	self.dense = self.dense[0:0]
	return self
}

func (self *FastSet) Slice() []uint {
	slice := make([]uint, 0, len(self.dense))
	for _, m := range self.dense {
		slice = append(slice, m)
	}
	return slice
}

func (self *FastSet) Add(i uint) *FastSet {
	if i >= uint(len(self.sparse)) {
		return self
	}
	if self.Has(i) {
		return self
	}
	n := uint(len(self.dense))
	if n == uint(cap(self.dense)) {
		panic("overflow")
	}
	self.dense = append(self.dense, i)
	self.sparse[i] = n
	return self
}

func (self *FastSet) Remove(i uint) *FastSet {
	if i >= uint(len(self.sparse)) {
		return self
	}
	if !self.Has(i) {
		return self
	}
	j := self.dense[len(self.dense)-1];
	self.dense[self.sparse[i]] = j;
	self.sparse[j] = self.sparse[i];
	self.dense = self.dense[0:len(self.dense)-1]
	return self
}

func (self *FastSet) Has(i uint) bool {
	return i < uint(len(self.sparse)) &&
	       self.sparse[i] < uint(len(self.dense)) &&
	       self.dense[self.sparse[i]] == i
}

func max(a, b uint) uint {
	if a > b {
		return a
	}
	return b
}

func (self *FastSet) Union(o *FastSet) *FastSet {
	set := NewFastSet(max(self.N(), o.N()))
	for _, i := range self.dense {
		set.Add(i)
	}
	for _, i := range o.dense {
		set.Add(i)
	}
	return set
}

func (self *FastSet) UnionInPlace(o *FastSet) *FastSet {
	for _, i := range o.dense {
		self.Add(i)
	}
	return self
}

func (self *FastSet) Intersect(o *FastSet) *FastSet {
	set := NewFastSet(max(self.N(), o.N()))
	for _, i := range self.dense {
		if o.Has(i) {
			set.Add(i)
		}
	}
	return set
}

func (self *FastSet) IntersectInPlace(o *FastSet) *FastSet {
	for _, i := range self.dense {
		if !o.Has(i) {
			self.Remove(i)
		}
	}
	return self
}

func (self *FastSet) Difference(o *FastSet) *FastSet {
	set := NewFastSet(self.N())
	for _, i := range self.dense {
		if !o.Has(i) {
			set.Add(i)
		}
	}
	return set
}

func (self *FastSet) DifferenceInPlace(o *FastSet) *FastSet {
	for _, i := range self.dense {
		if o.Has(i) {
			self.Remove(i)
		}
	}
	return self
}

func (self *FastSet) Complement() *FastSet {
	set := NewFastSet(self.N())
	for i := uint(0); i < uint(cap(self.dense)); i++ {
		if !self.Has(i) {
			set.Add(i)
		}
	}
	return set
}

func (self *FastSet) String() string {
	members := make([]string, 0, len(self.dense))
	for _, m := range self.dense {
		members = append(members, fmt.Sprintf("%d", m))
	}
	return "{" + strings.Join(members, ", ") + "}"
}


