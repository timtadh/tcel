package table

import (
	"fmt"
)

type SymbolTable struct {
	symbols []map[string]interface{}
}

func NewSymbolTable() *SymbolTable {
	t := &SymbolTable{
		symbols: make([]map[string]interface{}, 0, 10),
	}
	t.Push()
	return t
}

func (self *SymbolTable) Push() {
	m := make(map[string]interface{})
	self.symbols = append(self.symbols, m)
}

func (self *SymbolTable) Pop() error {
	if len(self.symbols) <= 1 {
		return fmt.Errorf("Cannot Pop base table")
	}
	self.symbols = self.symbols[:len(self.symbols)-1]
	return nil
}

func (self *SymbolTable) Depth() int {
	return len(self.symbols) - 1
}

func (self *SymbolTable) Get(sym string) interface{} {
	for i := len(self.symbols) - 1; i >= 0; i-- {
		if e, has := self.symbols[i][sym]; has {
			return e
		}
	}
	return nil
}

func (self *SymbolTable) Put(name string, e interface{}) {
	self.symbols[len(self.symbols)-1][name] = e
}

func (self *SymbolTable) TopHas(name string) bool {
	_, has := self.symbols[len(self.symbols)-1][name]
	return has
}

func (self *SymbolTable) Has(name string) bool {
	return self.Get(name) != nil
}

