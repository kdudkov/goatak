package cot

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
	Nodes   []*Node    `xml:",any"`
}

func NewXmlDetails() *Node {
	return NewNode("detail", nil)
}

func NewNode(name string, attrs map[string]string) *Node {
	n := &Node{XMLName: xml.Name{Local: name}}

	for k, v := range attrs {
		if v != "" {
			n.Attrs = append(n.Attrs, xml.Attr{Name: xml.Name{Local: k}, Value: v})
		}
	}

	return n
}

func DetailsFromString(s string) (*Node, error) {
	x := new(Node)
	var b []byte

	if strings.HasPrefix(s, "<Detail>") || strings.HasPrefix(s, "<detail>") {
		b = []byte(s)
	} else {
		b = []byte("<detail>" + s + "</detail>")
	}

	buf := bytes.NewBuffer(b)
	err := xml.NewDecoder(buf).Decode(x)
	return x, err
}

func (n *Node) AddPpLink(uid, typ, callsign string) {
	params := make(map[string]string)
	if uid != "" {
		params["uid"] = uid
	}
	if typ != "" {
		params["type"] = typ
	}
	if callsign != "" {
		params["parent_callsign"] = callsign
	}
	//params["production_time"] = prodTime.UTC().Format(time.RFC3339)
	params["relation"] = "p-p"
	n.AddChild("link", params, "")
}

func (n *Node) AsXMLString() string {
	b := bytes.Buffer{}
	xml.NewEncoder(&b).Encode(n)
	s := b.String()
	if len(s) > 17 {
		return s[len("<detail>") : len(s)-len("</detail>")]
	} else {
		return ""
	}
}

func (n *Node) String() string {
	if len(n.Nodes) == 0 {
		return "*empty*"
	}

	s := new(bytes.Buffer)
	for _, n := range n.Nodes {
		n.print(s, "")
	}
	return s.String()
}

func (n *Node) GetDest() []string {
	r := make([]string, 0)

	marti := n.GetFirst("marti")
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

func (n *Node) RemoveTags(tags ...string) {
	if n == nil {
		return
	}

	newNodes := make([]*Node, 0)
	for _, x := range n.Nodes {
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
	n.Content = ""
	n.Nodes = newNodes
}

func (n *Node) GetFirst(name string) *Node {
	if n == nil {
		return nil
	}
	for _, n := range n.Nodes {
		if n.XMLName.Local == name {
			return n
		}
	}
	return nil
}

func (n *Node) Has(name string) bool {
	return n.GetFirst(name) != nil
}

func (n *Node) GetAll(name string) []*Node {
	if n == nil {
		return nil
	}

	res := make([]*Node, 0)

	for _, nn := range n.Nodes {
		if nn.XMLName.Local == name {
			res = append(res, nn)
		}
	}

	return res
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

func (n *Node) GetText() string {
	if n == nil {
		return ""
	}

	return n.Content
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

func (n *Node) AddChild(name string, params map[string]string, text string) *Node {
	if n == nil {
		return nil
	}

	nn := &Node{XMLName: xml.Name{Local: name}}

	for k, v := range params {
		if v != "" {
			nn.Attrs = append(nn.Attrs, xml.Attr{Name: xml.Name{Local: k}, Value: v})
		}
	}

	if text != "" {
		nn.Content = text
	}
	n.Nodes = append(n.Nodes, nn)
	return nn
}
