package types

import (
	"fmt"
	"strings"
)


type Type interface {
	String() string
	Equals(Type) bool
}

type Primative string

type Function struct {
	Parameters []Type
	ParamNames []string
	Returns Type
}

var Unit Primative = "unit"
var String Primative = "string"
var Float Primative = "float"
var Int Primative = "int"

var Primatives []Primative = []Primative{
	Unit, String, Float, Int,
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

func (self *Function) String() string {
	params := make([]string, 0, len(self.Parameters))
	for _, param := range self.Parameters {
		params = append(params, param.String())
	}
	return fmt.Sprintf("fn(%v)%v", strings.Join(params, ", "), self.Returns)
}

