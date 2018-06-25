// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package fit

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unsafe"
)

const (
	magic      = 0xd00dfeed
	begin_node = 0x1 // Start node: full name
	end_node   = 0x2 // End node
	prop       = 0x3 // Property
	nop        = 0x4 // nop
	end        = 0x9 // End of fdt

	headerVer      = 17
	headerLastComp = 16
	headerLen      = 40
)

type header struct {
	Magic        uint32
	TotalSize    uint32 // total size of DT block
	OffDtStruct  uint32 // offset to structure
	OffDtStrings uint32 // offset to strings
	OffMemRsvmap uint32 // offset to memory reserve map

	Version               uint32
	LastCompatibleVersion uint32

	// version 2 fields below
	BootCpuidPhys uint32 // Which physical CPU id we're
	// booting on
	// version 3 fields below
	SizeDtStrings uint32 // size of the strings block

	// version 17 fields below
	SizeDtStruct uint32 // size of the structure block
}

func (h *header) String() string {
	return fmt.Sprintf("magic: 0x%x, version %d %d, total size: 0x%x, offset struct 0x%x strings 0x%x mem-reserve-map 0x%x",
		h.Magic, h.Version, h.LastCompatibleVersion,
		h.TotalSize, h.OffDtStruct, h.OffDtStrings, h.OffMemRsvmap)
}

type Node struct {
	Name       string
	Depth      int
	Properties map[string][]byte
	Children   map[string]*Node
}

type Tree struct {
	header
	Debug          bool
	IsLittleEndian bool
	RootNode       *Node
}

var defaultTree Tree

func init() {
	_ = defaultTree.ParseKernel()
}

func DefaultTree() (t *Tree) {
	d := defaultTree // Deliberate copy - caller can modify
	if d.RootNode != nil && len(d.RootNode.Properties) != 0 &&
		len(d.RootNode.Children) != 0 {
		return &d
	}
	return nil
}

func (n *Node) String() (s string) {
	if n == nil {
		return "nil"
	}
	s = fmt.Sprintf("%*s%s: ", 2*n.Depth, " ", n.Name)
	for name, value := range n.Properties {
		s += fmt.Sprintf("\n%*s%s = %q", 2*(1+n.Depth), " ", name,
			value)
	}
	for _, c := range n.Children {
		s += fmt.Sprintf("\n%s", c)
	}
	return
}

func (t *Tree) String() string { return t.RootNode.String() }

func (t *Tree) getCell(b []byte, i int) (value int, r int) {
	value = int(t.PropUint32(b[i:]))
	r = i + 4
	return
}

func (t *Tree) getString(b []byte, offset int) string {
	o := int(t.OffDtStrings) + offset
	l := bytes.IndexByte(b[o:], 0)
	return string(b[o : o+l])
}

func align(x int, align int) int {
	return (x + align - 1) & ^(align - 1)
}

// Read FDT header from blob and convert into
// right endian
func (t *Tree) readHeader(buf []byte) {
	var err error

	fh := bytes.NewReader(buf)
	if t.IsLittleEndian {
		err = binary.Read(fh, binary.LittleEndian, &t.header)
	} else {
		err = binary.Read(fh, binary.BigEndian, &t.header)
	}
	if err != nil {
		fmt.Println("binary.ReadFdtHeader failed:", err)
	}
}

func (t *Tree) Parse(buf []byte) (err error) {
	h := &t.header

	// Parse blob header
	t.readHeader(buf)
	if t.Debug {
		fmt.Printf("%+v\n", h)
	}

	// Walk thru nodes until done
	cur := int(h.OffDtStruct)
	stack := []*Node{}
	for {
		var tag int
		tag, cur = t.getCell(buf, cur)
		if tag == end {
			break
		}

		switch tag {
		case begin_node:
			n := &Node{}
			nameLen := bytes.IndexByte(buf[cur:], 0)
			n.Name = "/"
			if nameLen > 0 {
				n.Name = string(buf[cur : cur+nameLen])
			}
			if t.Debug {
				fmt.Printf("BEGIN_NODE: `%s'\n", n.Name)
			}
			cur = align(cur+nameLen+1, 4)
			stack = append(stack, n)
			n.Depth = len(stack)
		case end_node:
			// pop node stack
			var l int
			if l = len(stack); l == 1 {
				t.RootNode = stack[0]
			} else {
				c := stack[l-1]
				p := stack[l-2]
				if p.Children == nil {
					p.Children = make(map[string]*Node)
				}
				p.Children[c.Name] = c
			}
			stack = stack[:l-1]
			if t.Debug {
				fmt.Println("END_NODE:")
			}
		case nop:
			if t.Debug {
				fmt.Println("NOP:")
			}
		case prop:
			var valueSize, nameOffset int
			valueSize, cur = t.getCell(buf, cur)
			nameOffset, cur = t.getCell(buf, cur)

			name := t.getString(buf, nameOffset)
			value := buf[cur : cur+valueSize]

			n := stack[len(stack)-1]
			if n.Properties == nil {
				n.Properties = make(map[string][]byte)
			}
			n.Properties[name] = value

			if t.Debug {
				fmt.Printf("PROP: %s = %v %q\n", name, value, string(value))
			}

			cur = align(cur+int(valueSize), 4)
		}
	}

	if len(stack) != 0 {
		err = errors.New("node stack not balanced")
	}

	return
}

func (n *Node) eachProp(propName string, propValue string, f func(n *Node, name string, value string)) {
	if len(propValue) > 0 {
		if value := n.Properties[propName]; strings.Contains(string(value), propValue) {
			f(n, propName, string(value))
		}
	} else if _, present := n.Properties[propName]; present {
		value := n.Properties[propName]
		f(n, propName, string(value))
	}

	for _, c := range n.Children {
		c.eachProp(propName, propValue, f)
	}
}

// Call user's function for each node with given property.
func (t *Tree) EachProperty(propName string, propValue string,
	f func(n *Node, name string, value string)) {
	t.RootNode.eachProp(propName, propValue, f)
}

// Recursive search for node named "nodeName" and when found run f()
func (n *Node) matchNode(nodeName string, f func(n *Node)) {
	if n.Name == nodeName {
		f(n)
	}
	for _, c := range n.Children {
		c.matchNode(nodeName, f)
	}
}

// Find node with specified name "nodeName" then run f() on it
func (t *Tree) MatchNode(nodeName string, f func(n *Node)) {
	t.RootNode.matchNode(nodeName, f)
}

// Find node of given name "nodeName"
func (n *Node) getNode(nodeName string) *Node {
	if n.Name == nodeName {
		return n
	}
	for _, c := range n.Children {
		cn := c.getNode(nodeName)
		if cn != nil {
			return cn
		}
	}
	return nil
}

// Run f() on every node from the starting node "n"
func (n *Node) eachNode(f func(n *Node)) {
	f(n)
	for _, c := range n.Children {
		c.eachNode(f)
	}
}

// Given a starting node name, descend that node applying f() along the way
func (t *Tree) EachNodeFrom(nodeName string, f func(n *Node)) {
	tn := t.RootNode.getNode(nodeName)
	if tn != nil {
		tn.eachNode(f)
	}
}

func (n *Node) eachRegexp(pattern *regexp.Regexp, f func(n *Node)) {
	for name := range n.Properties {
		if pattern.MatchString(name) {
			f(n)
			break
		}
	}
	for _, c := range n.Children {
		c.eachRegexp(pattern, f)
	}
}

// As abote but matching property name as a regexp.
func (t *Tree) EachPropertyMatching(pattern string, f func(n *Node)) {
	re := regexp.MustCompile(pattern)
	t.RootNode.eachRegexp(re, f)
}

// Parses property value as 32 bit integer.
func (t *Tree) PropUint32(b []byte) (value uint32) {
	if t.IsLittleEndian {
		value = binary.LittleEndian.Uint32(b)
	} else {
		value = binary.BigEndian.Uint32(b)
	}
	return
}

// Property value as slice of 32 bit integers.
func (t *Tree) PropUint32Slice(b []byte) (value []uint32) {
	value = make([]uint32, len(b)/4)
	for i := range value {
		value[i] = t.PropUint32(b[i*4:])
	}
	return
}

// Property value as go string.
func (t *Tree) PropString(b []byte) (s string) {
	v := t.PropStringSlice(b)
	return v[0]
}

// Property value as go string slice.
func (t *Tree) PropStringSlice(b []byte) (s []string) {
	return strings.Split(string(b), "\x00")
}

// Write support
func (t *Tree) alignTo(b []byte, align int) []byte {
	for len(b)&(align-1) != 0 {
		b = append(b, 0)
	}
	return b
}

func (t *Tree) PropUint32ToSlice(v uint32) (r []byte) {
	r = make([]byte, 4)
	if t.IsLittleEndian {
		binary.LittleEndian.PutUint32(r, v)
	} else {
		binary.BigEndian.PutUint32(r, v)
	}
	return r
}

func (t *Tree) PropUint64ToSlice(v uint64) (r []byte) {
	r = make([]byte, 8)
	if t.IsLittleEndian {
		binary.LittleEndian.PutUint64(r, v)
	} else {
		binary.BigEndian.PutUint64(r, v)
	}
	return r
}

func (t *Tree) putCell(b []byte, v uint32) []byte {
	return append(b, t.PropUint32ToSlice(v)...)
}

func (t *Tree) putCellUint64(b []byte, v uint64) []byte {
	return append(b, t.PropUint64ToSlice(v)...)
}

func (t *Tree) putNode(b []byte, s []byte, n *Node) (bOut []byte, sOut []byte) {
	b = t.putCell(b, begin_node)

	if n.Name != "/" {
		b = append(b, []byte(n.Name)...)
	}
	b = append(b, 0)
	b = t.alignTo(b, 4)

	for name, value := range n.Properties {
		b = t.putCell(b, prop)
		b = t.putCell(b, uint32(len(value)))
		b = t.putCell(b, uint32(len(s)))
		b = append(b, []byte(value)...)
		s = append(s, []byte(name)...)
		s = append(s, 0)
		b = t.alignTo(b, 4)
	}

	for _, c := range n.Children {
		b, s = t.putNode(b, s, c)
	}

	b = t.putCell(b, end_node)

	return b, s
}

func (t *Tree) putUint32(w *uint32, v uint32) {
	r := t.PropUint32ToSlice(v)
	*w = *(*uint32)(unsafe.Pointer(&r[0]))
}

func (t *Tree) FlattenTreeToSlice() []byte {
	h := make([]byte, headerLen) // Header block
	m := make([]byte, 8*2)       // Dummy memory reservation block
	b := make([]byte, 0)         // Structure block
	s := make([]byte, 0)         // String block

	b, s = t.putNode(b, s, t.RootNode) // Build structure and string block
	b = t.putCell(b, end)              // End the tree
	s = t.alignTo(s, 4)                // Align string block

	hdr := (*header)(unsafe.Pointer(&h[0])) // Get header pointer

	t.putUint32(&hdr.Magic, magic)
	t.putUint32(&hdr.TotalSize, uint32(headerLen+len(m)+len(b)+len(s)))
	t.putUint32(&hdr.OffDtStruct, uint32(headerLen+len(m)))
	t.putUint32(&hdr.OffDtStrings, uint32(headerLen+len(m)+len(b))) // offset to strings
	t.putUint32(&hdr.OffMemRsvmap, headerLen)                       // offset to memory reserve map

	t.putUint32(&hdr.Version, headerVer)
	t.putUint32(&hdr.LastCompatibleVersion, headerLastComp)

	t.putUint32(&hdr.BootCpuidPhys, 0)              // Which physical CPU id we're
	t.putUint32(&hdr.SizeDtStrings, uint32(len(s))) // size of the strings block
	t.putUint32(&hdr.SizeDtStruct, uint32(len(b)))  // size of the structure block

	h = append(h, m...)
	h = append(h, b...)
	h = append(h, s...)

	return h
}
