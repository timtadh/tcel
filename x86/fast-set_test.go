package x86

import "testing"


func TestAdd(t *testing.T) {
	set := NewFastSet(25)
	set.Add(5).Add(7).Add(2).Add(9)
	s := set.String()
	e := "{5, 7, 2, 9}"
	if s != e {
		t.Errorf("expected %v got %v", e, s)
	}
}

func TestRemove(t *testing.T) {
	set := NewFastSet(25)
	set.Add(5).Add(7).Add(2).Add(9).Remove(2)
	s := set.String()
	se := "{5, 7, 9}"
	if s != se {
		t.Errorf("expected %v got %v", se, s)
	}
	e := []uint{5, 7, 9}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
	set.Remove(9).Remove(5)
	if !set.Has(7) {
		t.Errorf("should have had 7 but did not %v", set)
	}
	if set.String() != "{7}" {
		t.Errorf("Had other things besides 7, %v", set)
	}
	set.Remove(7)
	if set.String() != "{}" {
		t.Errorf("Was not empty, %v", set)
	}
}

func TestClear(t *testing.T) {
	set := NewFastSet(25)
	set.Add(5).Add(7).Add(2).Add(9)
	e := []uint{5, 7, 2, 9}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
	set.Clear()
	for i := uint(0); i < 25; i++ {
		if set.Has(i) {
			t.Errorf("had %d but should not have had it, %v", i, set)
		}
	}
}

func TestHas(t *testing.T) {
	set := NewFastSet(25)
	set.Add(5).Add(7).Add(2).Add(9)
	e := []uint{5, 7, 2, 9}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
}

func TestNotHas(t *testing.T) {
	set := NewFastSet(25)
	set.Add(5).Add(7).Add(2).Add(9)
	e := map[uint]bool{5:true, 7:true, 2:true, 9:true}
	for i := uint(0); i < 25; i++ {
		if _, has := e[i]; !has && set.Has(i) {
			t.Errorf("had %d but should not have had it, %v", i, set)
		}
	}
}

func TestUnion(t *testing.T) {
	set := NewFastSet(25).Add(5).Add(7).Union(NewFastSet(32).Add(2).Add(9))
	e := []uint{5, 7, 2, 9}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
}

func TestUnionInPlace(t *testing.T) {
	set := NewFastSet(25).Add(5).Add(7).UnionInPlace(NewFastSet(32).Add(2).Add(9))
	e := []uint{5, 7, 2, 9}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
}

func TestIntersectInPlace(t *testing.T) {
	set := NewFastSet(25).Add(5).Add(7).Add(15).Add(20).IntersectInPlace(
			NewFastSet(32).Add(7).Add(9).Add(31).Add(20))
	e := []uint{7, 20}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
}

func TestDifferenceInPlace(t *testing.T) {
	set := NewFastSet(25).Add(5).Add(7).Add(15).Add(20).DifferenceInPlace(
			NewFastSet(32).Add(7).Add(9).Add(31).Add(20))
	e := []uint{5, 15}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
	}
}

func TestComplement(t *testing.T) {
	set := NewFastSet(25).Add(5).Add(7).Add(15).Add(20).Intersect(
			NewFastSet(32).Add(7).Add(9).Add(31).Add(20))
	nset := set.Complement()
	e := []uint{7, 20}
	m := map[uint]bool{7:true, 20:true}
	for _, i := range e {
		if !set.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, set)
		}
		if nset.Has(i) {
			t.Errorf("had %d but should not have, %v", i, nset)
		}
	}
	for i := uint(0); i < 25; i++ {
		if _, has := m[i]; !has && set.Has(i) {
			t.Errorf("had %d but should not have, %v", i, set)
		}
		if _, has := m[i]; !has && !nset.Has(i) {
			t.Errorf("should have had %d but did not, %v", i, nset)
		}
	}
}

