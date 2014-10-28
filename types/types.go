package types

import (
	"fmt"
	"strings"
)


type Type interface {
	String() string
	Equals(Type) bool
	Empty() interface{}
}

type Primative string

type Function struct {
	Parameters []Type
	Returns Type
}

type Array struct {
	Base Type
}

var Label Primative = "label"
var Unit Primative = "unit"
var String Primative = "string"
var Float Primative = "float"
var Int Primative = "int"
var Boolean Primative = "boolean"

var Primatives []Primative = []Primative{
	Unit, String, Float, Int, Boolean,
}

func (self Primative) String() string {
	return string(self)
}

func (self Primative) Equals(o Type) bool {
	if t, ok := o.(Primative); ok {
		return string(t) == string(self)
	}
	return false
}

func (self Primative) Empty() interface{} {
	switch string(self) {
	case "string": return ""
	case "float": return float64(0.0)
	case "int": return int64(0)
	case "boolean": return false
	}
	panic(fmt.Errorf("Cannot construct and empty %v", self))
}

func (self *Function) Equals(o Type) bool {
	t, ok := o.(*Function)
	if !ok {
		return false
	}
	if len(self.Parameters) != len(t.Parameters) {
		return false
	}
	if !self.Returns.Equals(t.Returns) {
		return false
	}
	for i, a := range self.Parameters {
		if !a.Equals(t.Parameters[i]) {
			return false
		}
	}
	return true
}

func (self *Function) Empty() interface{} {
	panic("cannot construct an empty function yet")
}

func (self *Function) String() string {
	params := make([]string, 0, len(self.Parameters))
	for _, param := range self.Parameters {
		params = append(params, param.String())
	}
	return fmt.Sprintf("fn(%v)%v", strings.Join(params, ", "), self.Returns)
}

func (self *Array) Equals(o Type) bool {
	t, ok := o.(*Array)
	if !ok {
		return false
	}
	return self.Base.Equals(t.Base)
}

func (self *Array) String() string {
	return fmt.Sprintf("[]%v", self.Base)
}

func (self *Array) Empty() interface{} {
	return make([]interface{}, 0)
}

