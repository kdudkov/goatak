package cot

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

type XMLDetails struct {
	node *Node
}

type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",innerxml"`
	Nodes   []*Node    `xml:",any"`
}

func NewXmlDetails() *XMLDetails {
	return &XMLDetails{node: &Node{
		XMLName: xml.Name{Local: "detail"},
		Attrs:   nil,
		Content: "",
		Nodes:   nil,
	}}
}
func DetailsFromString(s string) (*XMLDetails, error) {
	x := &XMLDetails{node: new(Node)}
	var b []byte

	if strings.HasPrefix(s, "<Detail>") || strings.HasPrefix(s, "<detail>") {
		b = []byte(s)
	} else {
		b = []byte("<detail>" + s + "</detail>")
	}

	buf := bytes.NewBuffer(b)
	err := xml.NewDecoder(buf).Decode(x.node)
	return x, err
}

func (x *XMLDetails) AsXMLString() string {
	b := bytes.Buffer{}
	xml.NewEncoder(&b).Encode(x.node)
	s := b.String()
	return s[len("<detail>") : len(s)-len("</detail>")]
}

func (x *XMLDetails) String() string {
	if x.node == nil || len(x.node.Nodes) == 0 {
		return "*empty*"
	}

	s := new(bytes.Buffer)
	for _, n := range x.node.Nodes {
		n.print(s, "")
	}
	return s.String()
}

func (x *XMLDetails) GetDest() []string {
	r := make([]string, 0)

	marti := x.GetFirstChild("marti")
	if marti == nil {
		return r
	}

	for _, n := range marti.Nodes {
		if n.XMLName.Local == "dest" {
			if c := n.GetAttr("callsign"); c != "" {
				r = append(r, c)
			}
		}
	}

	return r
}

func (x *XMLDetails) RemoveTags(tags ...string) {
	if x == nil {
		return
	}

	newNodes := make([]*Node, 0)
	for _, x := range x.node.Nodes {
		found := false
		for _, t := range tags {
			if t == x.XMLName.Local {
				found = true
				break
			}
		}
		if !found {
			newNodes = append(newNodes, x)
		}
	}
	x.node.Content = ""
	x.node.Nodes = newNodes
}

func (x *XMLDetails) GetFirstChild(name string) *Node {
	node := x.node

	for _, s := range strings.Split(name, "/") {
		found := false
		for _, n := range node.Nodes {
			if n.XMLName.Local == s {
				node = n
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	return node
}

func (n *Node) GetFirstChild(name string) *Node {
	for _, n := range n.Nodes {
		if n.XMLName.Local == name {
			return n
		}
	}
	return nil
}

func (x *XMLDetails) HasChild(name string) bool {
	return x.GetFirstChild(name) != nil
}

func (x *XMLDetails) GetChildValue(name string) (string, bool) {
	for _, n := range x.node.Nodes {
		if n.XMLName.Local == name {
			return n.Content, true
		}
	}

	return "", false
}

func (n *Node) GetAttr(name string) string {
	if n == nil {
		return ""
	}

	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}

	return ""
}

func (n *Node) print(s *bytes.Buffer, prefix string) {
	s.WriteString(prefix + n.XMLName.Local)
	if len(n.Attrs) > 0 {
		s.WriteString(" [")
		for i, a := range n.Attrs {
			if i > 0 {
				s.WriteRune(',')
			}
			s.WriteString(fmt.Sprintf("%s=\"%s\"", a.Name.Local, a.Value))
		}
		s.WriteString("]")
	}
	s.WriteByte('\n')

	if n.Content != "" {
		s.WriteString(prefix + "> ")
		s.WriteString(n.Content)
		s.WriteByte('\n')
	}
	for _, n := range n.Nodes {
		n.print(s, prefix+"    ")
	}
}

func (n *Node) AddChild(name string, params map[string]string) *Node {
	nn := &Node{XMLName: xml.Name{Local: name}}

	for k, v := range params {
		nn.Attrs = append(nn.Attrs, xml.Attr{Name: xml.Name{Local: k}, Value: v})
	}

	n.Nodes = append(n.Nodes, nn)
	return nn
}

func (n *Node) AddChildWithContext(name string, params map[string]string, text string) *Node {
	nn := &Node{XMLName: xml.Name{Local: name}}

	for k, v := range params {
		nn.Attrs = append(nn.Attrs, xml.Attr{Name: xml.Name{Local: k}, Value: v})
	}

	nn.Content = text
	n.Nodes = append(n.Nodes, nn)
	return nn
}
