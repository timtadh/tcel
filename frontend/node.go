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

type Node struct {
	Label    string
	Value    interface{}
	Type     types.Type
	Children []*Node
	Location *SourceLocation
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
		Location: &SourceLocation{
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

func (self *Node) AddLeftMostKid(kid *Node, names ...string) *Node {
	if kid == nil {
		return self
	}
	if len(self.Get(0).Children) > 0 {
		if len(names) == 0 {
			self.Get(0).AddLeftMostKid(kid)
			return self
		} else {
			for _, name := range names {
				if self.Get(0).Label == name {
					self.Get(0).AddLeftMostKid(kid, names...)
					return self
				}
			}
		}
	}
	kids := self.Children
	self.Children = []*Node{kid}
	self.Children = append(self.Children, kids...)
	return self
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
			s = fmt.Sprintf("%s %s: (%d-%d)-(%d-%d)", s,
				n.Location.Filename, n.Location.StartLine, n.Location.StartColumn, n.Location.EndLine, n.Location.EndColumn)
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

