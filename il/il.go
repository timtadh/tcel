package il

import (
	"fmt"
	"strings"
	"reflect"
)

import (
	"github.com/timtadh/tcel/types"
)

type Functions map[string]*Func

func (funcs Functions) String() string {
	lines := make([]string, 0, 100)
	for name, fn := range funcs {
		lines = append(lines, fmt.Sprintf("%v %v", name, fn.Type))
		scope := make([]string, 0, len(fn.StaticScope))
		for _, x := range fn.StaticScope {
			scope = append(scope, x.Name)
		}
		lines = append(lines, fmt.Sprintf("  scope [%v]", strings.Join(scope, ", ")))
		for _, blk := range fn.BlockList {
			lines = append(lines, fmt.Sprintf("  %v", blk))
			for _, i := range blk.Insts {
				lines = append(lines, fmt.Sprintf("    %v", i))
			}
			lines = append(lines, "")
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type Func struct {
	Name      string
	Type      *types.Function
	Blocks    map[string]*Block
	BlockList []*Block
	Registers []*Register
	Scope     uint16
	StaticScope []*Func
	entry     *Block
	next_blk  int
	g         *ilGen
}

func (self *Func) Entry() *Block {
	return self.entry
}

func (self *Func) AddBlock(b *Block) {
	self.Blocks[b.Name] = b
	self.BlockList = append(self.BlockList, b)
}

func (self *Func) AddNewBlock() *Block {
	b := self.NewBlock()
	self.AddBlock(b)
	return b
}

func (self *Func) NewRegister(t types.Type) *Operand {
	o := &Operand{
		Type: t,
		Value: &Register{
			Id:    uint32(len(self.Registers)),
			Type:  t,
			Scope: self.Scope,
		},
	}
	self.Registers = append(self.Registers, o.Value.(*Register))
	return o
}

type Block struct {
	Name   string
	Insts  InstSlice
	Next   []*Block
	Prev   []*Block
	closed bool
	Fn     *Func
}

func (f *Func) NewBlock() *Block {
	name := fmt.Sprintf("%s-b-%d", f.Name, f.next_blk)
	f.next_blk++
	return &Block{
		Name:  name,
		Insts: make([]*Inst, 0, 10),
		Fn:    f,
	}
}

func (self *Block) Close() {
	self.closed = true
}

func (self *Block) Closed() bool {
	return self.closed
}

func (self *Block) Add(i *Inst) {
	if self.closed {
		panic("adding instruction to closed block")
	}
	self.Insts = append(self.Insts, i)
}

func (self *Block) Link(o *Block) {
	if self.closed {
		panic("linking a closed block")
	}
	self.Next = append(self.Next, o)
	o.Prev = append(o.Prev, self)
}

func (self *Block) J(o *Block) {
	self.Link(o)
	self.Add(NewInst(Ops["J"], Jump(o), &UNIT, &UNIT))
}

func (self *Block) String() string {
	next := make([]string, 0, len(self.Next))
	for _, n := range self.Next {
		next = append(next, n.Name)
	}
	prev := make([]string, 0, len(self.Prev))
	for _, n := range self.Prev {
		prev = append(prev, n.Name)
	}
	return fmt.Sprintf(
		"%v prev:%v next:%v",
		self.Name,
		"{"+strings.Join(prev, ",")+"}",
		"{"+strings.Join(next, ",")+"}",
	)
}

type Inst struct {
	Op OpCode
	A  *Operand
	B  *Operand
	R  *Operand
}

type InstSlice []*Inst

func NewInst(op OpCode, a, b, r *Operand) *Inst {
	return &Inst{Op: op, A: a, B: b, R: r}
}

func (self *Inst) String() string {
	if self.A.Equals(&UNIT) && self.B.Equals(&UNIT) && self.R.Equals(&UNIT) {
		return fmt.Sprintf("%-4v", self.Op)
	} else if self.A.Equals(&UNIT) && self.B.Equals(&UNIT) && !self.R.Equals(&UNIT) {
		return fmt.Sprintf("%-4v %-31v %v", self.Op, "", self.R)
	} else if !self.A.Equals(&UNIT) && self.B.Equals(&UNIT) && self.R.Equals(&UNIT) {
		return fmt.Sprintf("%-4v %v", self.Op, self.A)
	} else if !self.A.Equals(&UNIT) && self.B.Equals(&UNIT) && !self.R.Equals(&UNIT) {
		return fmt.Sprintf("%-4v %-31v %v", self.Op, self.A, self.R)
	} else if !self.A.Equals(&UNIT) && !self.B.Equals(&UNIT) && self.R.Equals(&UNIT) {
		return fmt.Sprintf("%-4v %-15v %v", self.Op, self.A, self.B)
	} else {
		return fmt.Sprintf("%-4v %-15v %-15v %v", self.Op, self.A, self.B, self.R)
	}
}

func (self InstSlice) String() string {
	lines := make([]string, 0, len(self))
	for _, i := range self {
		lines = append(lines, fmt.Sprint(i))
	}
	return strings.Join(lines, "\n")
}

type OpCode uint

var OpNames = make(map[OpCode]string)
var Ops = map[string]OpCode{
	"INVALID": 0,
	"IMM":     1,
	"MV":      2,
	"ADD":     3,
	"SUB":     4,
	"MUL":     5,
	"DIV":     6,
	"MOD":     7,
	"CALL":    8,
	"PRM":     9,
	"RTRN":    10,
	"EXIT":    11,
	"NOP":     12,
	"J":       13,
	"IFEQ":    14,
	"IFNE":    15,
	"IFLT":    16,
	"IFLE":    17,
	"IFGT":    18,
	"IFGE":    19,
	"NEW":     20, // takes size in bytes
	"GET":     21, // takes a mem buf, (an offset, length pair), and a destination
	"PUT":     22, // takes an operand, (an offset, length pair), and a mem buf
	"SIZE":    23, // takes a mem buf
}

func init() {
	for k, v := range Ops {
		OpNames[v] = k
	}
}

func (op OpCode) String() string {
	return OpNames[op]
}

type Operand struct {
	Type  types.Type
	Value Value
}

var UNIT = Operand{types.Unit, &UnitValue{}}

func (self *Operand) Equals(o *Operand) bool {
	return self.Type.Equals(o.Type) && self.Value.Equals(o.Value)
}

func (self *Operand) String() string {
	if self.Equals(&UNIT) {
		return ""
	}
	return fmt.Sprintf("%v:%v", self.Value, self.Type)
}

func (self *Operand) Reg() bool {
	_, reg := self.Value.(*Register)
	return reg
}

type Value interface {
	Equals(Value) bool
	String() string // serialization
}

type Register struct {
	Type  types.Type
	Id    uint32
	Scope uint16
}

func (self *Register) String() string {
	return fmt.Sprintf("R{%d,%d}", self.Id, self.Scope)
}

func (self *Register) Equals(v Value) bool {
	if o, is := v.(*Register); is {
		return self.Id == o.Id && self.Scope == o.Scope && self.Type.Equals(o.Type)
	}
	return false
}

type CallArgs struct {
	Operands []*Operand
}

func Params(prms []*Operand) *Operand {
	parts := make([]types.Type, 0, len(prms))
	for _, prm := range prms {
		parts = append(parts, prm.Type)
	}
	return &Operand{
		Type: types.Tuple(parts),
		Value: &CallArgs{Operands: prms},
	}
}

func (self *CallArgs) Equals(v Value) bool {
	if o, is := v.(*CallArgs); is {
		if len(self.Operands) != len(o.Operands) {
			return false
		}
		for i, a := range self.Operands {
			if !a.Equals(o.Operands[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (self *CallArgs) String() string {
	opers := make([]string, 0, len(self.Operands))
	for _, o := range self.Operands {
		opers = append(opers, fmt.Sprintf("%v", o))
	}
	return "(" + strings.Join(opers, ",") + ")"
}

type UnitValue struct{}

func (self *UnitValue) String() string {
	return "unit"
}

func (self *UnitValue) Equals(v Value) bool {
	if _, is := v.(*UnitValue); is {
		return true
	}
	return false
}

type CallTarget struct {
	Fn  *Func
}

func Call(fn *Func) *Operand {
	return &Operand{
		Type:  types.Label,
		Value: &CallTarget{Fn:fn},
	}
}

func (self *CallTarget) String() string {
	return self.Fn.Name
}

func (self *CallTarget) Equals(v Value) bool {
	if o, is := v.(*CallTarget); is {
		return o.Fn.Name == self.Fn.Name
	}
	return false
}

type JumpTarget struct {
	Blk *Block
}

func Jump(blk *Block) *Operand {
	return &Operand{
		Type:  types.Label,
		Value: &JumpTarget{Blk:blk},
	}
}

func (self *JumpTarget) String() string {
	return self.Blk.Name
}

func (self *JumpTarget) Equals(v Value) bool {
	if o, is := v.(*JumpTarget); is {
		return o.Blk.Name == self.Blk.Name
	}
	return false
}

type NativeTarget struct {
	Label string
}

func (self *NativeTarget) String() string {
	return self.Label
}

func (self *NativeTarget) Equals(v Value) bool {
	if o, is := v.(*NativeTarget); is {
		return o.Label == self.Label
	}
	return false
}

type Constant struct {
	Value interface{}
}

func Const(c interface{}) *Operand {
	var t types.Type
	switch x := c.(type) {
	case int:
		t = types.Int
		c = int64(x)
	case int64: t = types.Int
	case float64: t = types.Float
	case string: t = types.String
	case bool: t = types.Boolean
	default: panic(fmt.Errorf("Unexpected constant value %v", c))
	}
	return &Operand{
		Type:  t,
		Value: &Constant{Value:c},
	}
}

func (self *Constant) String() string {
	return fmt.Sprintf("%v", self.Value)
}

func (self *Constant) Equals(v Value) bool {
	if o, is := v.(*Constant); is {
		return reflect.DeepEqual(self.Value, o.Value)
	}
	return false
}

type OffsetLength struct {
	Offset int
	Length int
}

func OffLen(offset, length int) *Operand {
	return &Operand{
		Type:  types.Tuple([]types.Type{types.Int, types.Int}),
		Value: &OffsetLength{offset, length},
	}
}

func (self *OffsetLength) String() string {
	return fmt.Sprintf("(%d,%d)", self.Offset, self.Length)
}

func (self *OffsetLength) Equals(v Value) bool {
	if o, is := v.(*OffsetLength); is {
		return self.Offset == o.Offset && self.Length == o.Length
	}
	return false
}
