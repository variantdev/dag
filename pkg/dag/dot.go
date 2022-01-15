package dag

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func (d *DAG) WriteDotTo(w io.Writer) error {
	fmt.Fprintln(w, "digraph DAG {\nrankdir=\"LR\"")

	ctx := &dot{
		writer:      w,
		nodeWritten: make(map[Key]bool),
		edgeWritten: make(map[edge]bool),
	}

	nodes := make([]Key, len(d.nodes))
	for i, n := range d.nodes {
		nodes[i] = n
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Less(nodes[j])
	})

	for _, n := range nodes {
		if err := ctx.writeNode(n, d.labels[n]); err != nil {
			return err
		}
	}

	for _, from := range nodes {
		outs, ok := d.outputs[from]
		if !ok {
			continue
		}
		tos := make([]Key, len(outs))
		i := 0
		for to, _ := range outs {
			tos[i] = to
			i += 1
		}
		sort.Slice(tos, func(i, j int) bool {
			return tos[i].Less(tos[j])
		})

		for _, to := range tos {
			if err := ctx.writeEdge(from, to); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintln(w, "}")
	return err
}

type dot struct {
	writer      io.Writer
	nodeWritten map[Key]bool
	edgeWritten map[edge]bool
}

type edge struct {
	from, to interface{}
}

func (c *dot) writeNode(v Key, labels map[string]bool) error {
	if c.nodeWritten[v] {
		return nil
	}
	c.nodeWritten[v] = true
	ls := []string{}
	if labels != nil {
		for l := range labels {
			ls = append(ls, l)
		}
		sort.Strings(ls)
	}

	var label string
	if len(ls) > 0 {
		label = fmt.Sprintf("{%s|{%s}}", v, strings.Join(ls, "|"))
	} else {
		label = fmt.Sprintf("{%s}", v)
	}

	_, err := fmt.Fprintf(c.writer, `%q [shape=record, label=%q]`+"\n", v, label)
	return err
}

func (c *dot) writeEdge(from, to Key) error {
	if c.edgeWritten[edge{from, to}] {
		return nil
	}
	c.edgeWritten[edge{from, to}] = true
	_, err := fmt.Fprintf(c.writer, `%q -> %q`+"\n", from, to)
	return err
}
