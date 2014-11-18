package x86

import (
	"github.com/timtadh/tcel/il"
)

type BlockAssignments []Assignments

type Assignments map[ILReg]X86Reg

type ILReg uint

type ILBlock struct {
	blk *il.Block
	regs []*il.Register
	rmap map[uint64]uint // from *il.Register to index in regs
}

type X86Reg string

var x86_regs []string = []string{
	"eax", "ebx", "ecx", "edx", "esi", "edi",
}

type assignment struct {
	x86reg string
	def uint
	reg uint
}

/*
func AllocateRegisters(blk *il.Block) BlockAssignments {
	b := NewILBlock(blk)
	live := b.LiveRegs()
	defs := b.ReachingDefs()

	free := make([]string, len(x86_regs))
	copy(free, x86_regs...)

	for x, i := range blk.Insts {
		
	}
}*/

func NewILBlock(blk *il.Block) *ILBlock {
	b := new(ILBlock)
	b.blk = blk
	b.rmap = make(map[uint64]uint)
	add := func(r *il.Register) {
		n := regnum(r)
		if _, has := b.rmap[n]; !has {
			b.rmap[n] = uint(len(b.regs))
			b.regs = append(b.regs, r)
		}
	}
	for _, i := range blk.Insts {
		if r, is := i.A.Value.(*il.Register); is {
			add(r)
		}
		if r, is := i.B.Value.(*il.Register); is {
			add(r)
		}
		if r, is := i.R.Value.(*il.Register); is {
			add(r)
		}
	}
	return b
}

func regnum(r *il.Register) uint64 {
	var reg uint64 = 0
	reg = uint64(r.Scope) << 32
	reg = reg | uint64(r.Id)
	return reg
}

/* Computes the live registers for each instruction. LiveRegs[i] correspondes
 * to which registers are live on *entry* to the instruction.
 */
func (blk *ILBlock) LiveRegs() [][]uint {

	live := make([][]uint, len(blk.blk.Insts))

	flow := NewFastSet(uint(len(blk.regs)))

	for x := len(blk.blk.Insts)-1; x >= 0; x-- {
		i := blk.blk.Insts[x]
		if r, is := i.R.Value.(*il.Register); is {
			flow.Remove(blk.rmap[regnum(r)])
		}
		if r, is := i.A.Value.(*il.Register); is {
			flow.Add(blk.rmap[regnum(r)])
		}
		if r, is := i.B.Value.(*il.Register); is {
			flow.Add(blk.rmap[regnum(r)])
		}
		live[x] = flow.Slice()
	}

	return live
}

/* Computes the reaching definitions at the start of each instruction. The
 * definition is the index of the defining instruction.
 */
func (blk *ILBlock) ReachingDef() [][]uint {

	reach_defs := make([][]uint, len(blk.blk.Insts))

	flow := NewFastSet(uint(len(blk.blk.Insts)))

	defines := func(x int) (uint, bool) {
		i := blk.blk.Insts[x]
		if r, is := i.R.Value.(*il.Register); is {
			return blk.rmap[regnum(r)], true
		}
		return 0, false
	}

	kills := make(map[uint]uint)
	for x := range blk.blk.Insts {
		if id, isreg := defines(x); isreg {
			if kill, has := kills[id]; has {
				flow.Remove(kill)
			}
			kills[id] = uint(x)
			flow.Add(uint(x))
			reach_defs[x] = flow.Slice()
		}
	}

	return reach_defs
}

