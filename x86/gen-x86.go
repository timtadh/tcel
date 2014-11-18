package x86

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/tcel/types"
	"github.com/timtadh/tcel/il"
)

var Lib string = `
#include <stdio.h>

extern void print_int(int);

void print_int(int i) {
	printf("%d\n", i);
}

`

func Generate(fns il.Functions) (string, error) {
	fmt.Println(fns)
	g := newGen()
	g.ProgramSetup()
	err := g.Functions(fns)
	if err != nil {
		fmt.Println(strings.Join(append(g.program, ""), "\n"))
		return "", err
	}
	return strings.Join(append(g.program, ""), "\n"), nil
}

type x86Gen struct {
	program []string
	rodata []string
	f       *frame
}

type frame struct {
	locs map[uint32]int
	fn *il.Func
}

func newGen() *x86Gen {
	return &x86Gen{
		program: make([]string, 0, 100),
	}
}

func (g *x86Gen) roAdd(line string) {
	g.rodata = append(g.rodata, line)
}

func (g *x86Gen) String(str string) string {
	name := fmt.Sprintf("string_%d", len(g.rodata))
	g.roAdd(fmt.Sprintf("%v:", name))
	g.roAdd(fmt.Sprintf(".string \"%d\"", str))
	return name
}

func (g *x86Gen) Add(line string) {
	g.program = append(g.program, fmt.Sprintf("    %v", line))
}

func (g *x86Gen) Direct(line string) {
	g.program = append(g.program, line)
}

func (g *x86Gen) Name(name string) string {
	return strings.Replace(name, "-", "_", -1)
}

func (g *x86Gen) Label(name string) {
	g.program = append(g.program, fmt.Sprintf("%v:", g.Name(name)))
}

func (g *x86Gen) Store(reg string, o *il.Operand) {
	g.Add(fmt.Sprintf("movl %%%v, %v", reg, g.Location(o)))
}

func (g *x86Gen) Load(o *il.Operand, reg string) {
	g.Add(fmt.Sprintf("movl %v, %%%v", g.Location(o), reg))
}

func (g *x86Gen) ProgramSetup() {
	g.Add("")
	g.Direct(".section .text")
	g.roAdd("")
	g.roAdd(".section .rodata")
}

func (g *x86Gen) Value(o *il.Operand) string {
	switch v := o.Value.(type) {
	case *il.CallTarget: return g.Name(v.Fn.Name)
	case *il.JumpTarget: return g.Name(v.Blk.Name)
	case *il.NativeTarget: return g.Name(v.Label)
	case *il.Constant: return g.ConstValue(v)
	}
	panic(fmt.Errorf("Can't gen a value of %v", o))
}

func (g *x86Gen) ConstValue(v *il.Constant) string {
	switch c := v.Value.(type) {
	case int64: return fmt.Sprintf("%v", c)
	case float64: panic(fmt.Errorf("not yet supported"))
	case string: return g.String(c)
	case bool: panic(fmt.Errorf("not yet supported"))
	}
	panic(fmt.Errorf("unexpected constant type %v, %T", v, v.Value))
}

func (g *x86Gen) Location(o *il.Operand) string {
	return g.location(o.Value.(*il.Register))
}

func (g *x86Gen) location(r *il.Register) string {
	if off, has := g.f.locs[r.Id]; r.Scope != g.f.fn.Scope || !has {
		panic(fmt.Errorf("could not get loc for %v in %v : %v", r, g.f.locs, g.f.fn))
	} else {
		return fmt.Sprintf("%d(%%ebp)", off)
	}
}

func (g *x86Gen) Functions(fns il.Functions) error {
	for _, fn := range fns {
		if err := g.Function(fn); err != nil {
			return err
		}
	}
	return nil
}

func (g *x86Gen) Function(fn *il.Func) error {
	g.Add("")
	g.Direct(fmt.Sprintf(".global %v", g.Name(fn.Name)))
	g.Direct(fmt.Sprintf(".type %v @function", g.Name(fn.Name)))
	g.Label(fn.Name)
	g.FnPush(fn)
	for _, blk := range fn.BlockList {
		if err := g.Block(blk); err != nil {
			return err
		}
	}
	return nil
}

func (g *x86Gen) FnPush(fn *il.Func) {
	g.f = &frame{
		fn: fn,
		locs: make(map[uint32]int),
	}
	g.Add("pushl %ebp")
	g.Add("movl %esp, %ebp")
	g.Add(fmt.Sprintf("subl $%d, %%esp", len(fn.Registers)*4))
	for i, r := range fn.Registers {
		g.f.locs[r.Id] = -4*i - 4
		g.Add(fmt.Sprintf("movl $0, %v", g.location(r)))
	}
}

func (g *x86Gen) FnPop() {
	g.Add("movl %ebp, %esp")
	g.Add("movl (%esp), %ebp")
	g.Add("addl $4, %esp")
	g.Add("ret")
}

func (g *x86Gen) Block(blk *il.Block) error {
	g.Label(blk.Name)
	for _, i := range blk.Insts {
		if err := g.Instruction(i); err != nil {
			return err
		}
	}
	return nil
}

func (g *x86Gen) Instruction(i *il.Inst) error {
	switch i.Op {
	case il.Ops["IMM"]: return g.IMM(i)
	case il.Ops["MV"]: return g.MV(i)
	case il.Ops["ADD"]: return g.ADD(i)
	case il.Ops["SUB"]: return g.SUB(i)
	case il.Ops["MUL"]: return g.MUL(i)
	case il.Ops["DIV"]: return g.DIV(i)
	case il.Ops["MOD"]: return g.MOD(i)
	case il.Ops["CALL"]: return g.CALL(i)
	case il.Ops["PRM"]: return g.PRM(i)
	case il.Ops["RTRN"]: return g.RTRN(i)
	case il.Ops["EXIT"]: return g.EXIT(i)
	case il.Ops["NOP"]: return g.NOP(i)
	case il.Ops["J"]: return g.J(i)
	case il.Ops["IFEQ"]: return g.IF(i)
	case il.Ops["IFNE"]: return g.IF(i)
	case il.Ops["IFLT"]: return g.IF(i)
	case il.Ops["IFLE"]: return g.IF(i)
	case il.Ops["IFGT"]: return g.IF(i)
	case il.Ops["IFGE"]: return g.IF(i)
	}
	return fmt.Errorf("unknown opcode %v", i)
}

func (g *x86Gen) IMM(i *il.Inst) error {
	g.Add(fmt.Sprintf("movl $%v, %v", g.Value(i.A), g.Location(i.R)))
	return nil
}

func (g *x86Gen) MV(i *il.Inst) error {
	if i.A.Reg() {
		g.Load(i.A, "eax")
		g.Store("eax", i.R)
	} else {
		g.Add(fmt.Sprintf("movl $%v, %v", g.Value(i.A), g.Location(i.R)))
	}
	return nil
}

func (g *x86Gen) ADD(i *il.Inst) error {
	g.Load(i.A, "eax")
	g.Load(i.B, "ebx")
	g.Add("addl %ebx, %eax")
	g.Store("eax", i.R)
	return nil
}

func (g *x86Gen) SUB(i *il.Inst) error {
	g.Load(i.A, "eax")
	g.Load(i.B, "ebx")
	g.Add("subl %ebx, %eax")
	g.Store("eax", i.R)
	return nil
}

func (g *x86Gen) MUL(i *il.Inst) error {
	g.Load(i.A, "eax")
	g.Load(i.B, "ebx")
	g.Add("mull %ebx")
	g.Store("eax", i.R)
	return nil
}

func (g *x86Gen) DIV(i *il.Inst) error {
	g.Load(i.A, "eax")
	g.Load(i.B, "ebx")
	g.Add("divl %ebx")
	g.Store("eax", i.R)
	return nil
}

func (g *x86Gen) MOD(i *il.Inst) error {
	g.Load(i.A, "eax")
	g.Load(i.B, "ebx")
	g.Add("divl %ebx")
	g.Store("edx", i.R)
	return nil
}

func (g *x86Gen) CALL(i *il.Inst) error {
	args := i.B.Value.(*il.CallArgs)
	for _, arg := range args.Operands {
		g.PUSH(arg)
	}
	if i.A.Reg() {
		g.Add(fmt.Sprintf("call *%v", g.Location(i.A)))
	} else {
		g.Add(fmt.Sprintf("call %v", g.Value(i.A)))
	}
	if !i.R.Type.Equals(types.Unit) {
		g.Store("eax", i.R)
	}
	if len(args.Operands) > 0 {
		g.Add(fmt.Sprintf("addl $%d, %%esp", 4*len(args.Operands)))
	}
	return nil
}

func (g *x86Gen) PUSH(o *il.Operand) {
	if o.Reg() {
		g.Add(fmt.Sprintf("pushl %v", g.Location(o)))
	} else {
		g.Add(fmt.Sprintf("pushl $%v", g.Value(o)))
	}
}

func (g *x86Gen) PRM(i *il.Inst) error {
	p := i.A.Value.(*il.Constant).Value.(int64)
	off := 4*int(p) + 8
	g.Add(fmt.Sprintf("movl %d(%%ebp), %%eax", off))
	g.Store("eax", i.R)
	return nil
}

func (g *x86Gen) RTRN(i *il.Inst) error {
	if i.A.Reg() {
		g.Load(i.A, "eax")
	}
	g.FnPop()
	return nil
}

func (g *x86Gen) EXIT(i *il.Inst) error {
	g.Add("pushl $0")
	g.Add("call exit")
	return nil
}

func (g *x86Gen) NOP(i *il.Inst) error {
	g.Add("nop")
	return nil
}

func (g *x86Gen) J(i *il.Inst) error {
	g.Add(fmt.Sprintf("jmp %v", g.Value(i.A)))
	return nil
}

func (g *x86Gen) IF(i *il.Inst) error {
	panic("unimplemented")
}

