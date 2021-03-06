package frontend

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/tcel/types"
)

type SourceLocation struct {
	Filename string
	StartLine, StartColumn, EndLine, EndColumn int
}

func (self *SourceLocation) String() string {
	return fmt.Sprintf("(%d-%d)-(%d-%d) in %v",
		self.StartLine, self.StartColumn, self.EndLine, self.EndColumn, self.Filename)
}
func (self *SourceLocation) Join(others ...*SourceLocation) (*SourceLocation, error) {
	if self == nil && len(others) > 0 {
		self = others[0]
		others = others[1:]
	} else if self == nil && len(others) == 0 {
		return nil, nil
	}
	name := self.Filename
	for _, o := range others {
		if name != o.Filename {
			return nil, fmt.Errorf("Cannot join to source locations from different files")
		}
	}

	min_start_line := self.StartLine
	min_start_col := self.StartColumn
	max_end_line := self.EndLine
	max_end_col := self.EndColumn

	for _, o := range others {
		if o.StartLine < min_start_line {
			min_start_line = o.StartLine
			min_start_col = o.StartColumn
		} else if o.StartLine == min_start_line && o.StartColumn < min_start_col {
			min_start_col = o.StartColumn
		}
		if o.EndLine > max_end_line {
			max_end_line = o.EndLine
			max_end_col = o.EndColumn
		} else if o.EndLine == max_end_line && o.EndColumn > max_end_col {
			max_end_col = o.EndColumn
		}
	}

	return &SourceLocation{
		Filename:name, StartLine:min_start_line, StartColumn:min_start_col,
		EndLine:max_end_line, EndColumn:max_end_col,
	}, nil
}

type Node struct {
	Label    string
	Value    interface{}
	Type     types.Type
	Children []*Node
	location *SourceLocation
}

func NewNode(label string) *Node {
	return &Node{
		Label:    label,
		Value:    nil,
		Children: make([]*Node, 0, 5),
	}
}

func NewValueNode(label string, value interface{}) *Node {
	return &Node{
		Label:    label,
		Value:    value,
		Children: make([]*Node, 0, 5),
	}
}

func NewTokenNode(tok *Token) *Node {
	return &Node{
		Label: Tokens[tok.Type],
		Value: tok.Value,
		location: &SourceLocation{
			Filename:tok.Filename,
			StartLine: tok.StartLine,
			StartColumn: tok.StartColumn,
			EndLine: tok.EndLine,
			EndColumn: tok.EndColumn,
		},
	}
}

func (self *Node) Leaf() bool {
	return len(self.Children) == 0
}

func (self *Node) AddKid(kid *Node) *Node {
	if kid != nil {
		self.Children = append(self.Children, kid)
	}
	return self
}

func (self *Node) PrependKid(kid *Node) *Node {
	kids := self.Children
	self.Children = []*Node{kid}
	self.Children = append(self.Children, kids...)
	return self
}

func (self *Node) Kid(label string) *Node {
	for _, kid := range self.Children {
		if kid.Label == label {
			return kid
		}
	}
	return nil
}

func (self *Node) Get(idx int) *Node {
	if idx < 0 {
		idx = len(self.Children) + idx
	}
	return self.Children[idx]
}

func (self *Node) WellTyped() bool {
	well_typed := self.Type != nil
	if well_typed {
		for _, c := range self.Children {
			well_typed = well_typed && c.WellTyped()
		}
	}
	return well_typed
}

func (self *Node) String() string {
	return fmt.Sprintf("(Node %v %d)", self.Label, len(self.Children))
}

func (self *Node) Location() *SourceLocation {
	if self == nil {
		return nil
	}
	var locs []*SourceLocation
	if self.location != nil {
		locs = append(locs, self.location)
	}
	for _, kid := range self.Children {
		kl := kid.Location()
		if kl != nil {
			locs = append(locs, kl)
		}
	}
	if len(locs) == 0 {
		return nil
	} else if len(locs) == 1 {
		return locs[0]
	}
	base := locs[0]
	others := locs[1:]
	l, e := base.Join(others...)
	if e != nil {
		panic(e)
	}
	return l
}

func (self *Node) Annotate(nodes []*Node) *Node {
	var locs []*SourceLocation
	for _, n := range nodes {
		l := n.Location()
		if l != nil {
			locs = append(locs, l)
		}
	}
	var e error
	self.location, e = self.Location().Join(locs...)
	if e != nil {
		panic(e)
	}
	return self
}

func (self *Node) Serialize(with_loc bool) string {
	fmt_node := func(n *Node) string {
		s := ""
		if n.Value != nil && n.Type != nil {
			s = fmt.Sprintf(
				"%d:%s,%v:%v",
				len(n.Children),
				n.Label,
				n.Value,
				n.Type,
			)
		} else if n.Value != nil {
			s = fmt.Sprintf(
				"%d:%s,%v",
				len(n.Children),
				n.Label,
				n.Value,
			)
		} else if n.Type != nil {
			s = fmt.Sprintf(
				"%d:%s:%v",
				len(n.Children),
				n.Label,
				n.Type,
			)
		} else {
			s = fmt.Sprintf(
				"%d:%s",
				len(n.Children),
				n.Label,
			)
		}
		if with_loc && n.Location != nil {
			s = fmt.Sprintf("%s:at %v", s, n.Location())
		}
		return s
	}
	walk := func(node *Node) (nodes []string) {
		type entry struct {
			n *Node
			i int
		}
		type node_stack []*entry
		pop := func(stack node_stack) (node_stack, *entry) {
			if len(stack) <= 0 {
				return stack, nil
			} else {
				return stack[0 : len(stack)-1], stack[len(stack)-1]
			}
		}

		stack := make(node_stack, 0, 10)
		stack = append(stack, &entry{node, 0})

		for len(stack) > 0 {
			var c *entry
			stack, c = pop(stack)
			if c.i == 0 {
				nodes = append(nodes, fmt_node(c.n))
			}
			if c.i < len(c.n.Children) {
				kid := c.n.Children[c.i]
				stack = append(stack, &entry{c.n, c.i + 1})
				stack = append(stack, &entry{kid, 0})
			}
		}
		return nodes
	}
	nodes := walk(self)
	return strings.Join(nodes, "\n")
}

