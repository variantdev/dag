package dag

import (
	"fmt"
	"sort"
	"strings"
)

type Option func(*DAG)

func Capacity(n int) Option {
	return func(g *DAG) {
		g.cap = n
	}
}

func Node(nodes ...string) Option {
	return func(g *DAG) {
		g.initNodes = nodes
	}
}

func Nodes(nodes []string) Option {
	return func(g *DAG) {
		g.initNodes = nodes
	}
}

type DAG struct {
	cap       int
	initNodes []string

	nodes     []string
	outputs   map[string]map[string]bool
	labels    map[string]map[string]bool
	numInputs map[string]int
}

func (g *DAG) AddNode(name string) bool {
	if _, ok := g.outputs[name]; ok {
		return false
	}
	g.nodes = append(g.nodes, name)
	g.outputs[name] = make(map[string]bool)
	g.numInputs[name] = 0
	return true
}

func New(opt ...Option) *DAG {
	g := &DAG{
		numInputs: make(map[string]int),
		outputs:   make(map[string]map[string]bool),
		labels:    make(map[string]map[string]bool),
	}

	for _, o := range opt {
		o(g)
	}

	g.nodes = make([]string, 0, g.cap)

	g.AddNodes(g.initNodes...)

	return g
}

func (g *DAG) AddNodes(names ...string) bool {
	for _, name := range names {
		if ok := g.AddNode(name); !ok {
			return false
		}
	}
	return true
}

func (g *DAG) AddEdge(from, to string) bool {
	m, ok := g.outputs[from]
	if !ok {
		return false
	}

	m[to] = true
	g.numInputs[to]++

	return true
}

func (g *DAG) AddDependency(sub string, dependencies ...string) bool {
	for _, d := range dependencies {
		if r := g.AddEdge(d, sub); !r {
			return false
		}
	}
	return true
}

func (g *DAG) AddDependencies(sub string, dependencies []string) bool {
	return g.AddDependency(sub, dependencies...)
}

func (g *DAG) AddLabel(sub string, labels ...string) {
	for _, d := range labels {
		m, ok := g.labels[sub]
		if !ok {
			m = map[string]bool{}
			g.labels[sub] = m
		}
		m[d] = true
	}
}

func (g *DAG) AddLabels(sub string, labels []string) {
	g.AddLabel(sub, labels...)
}

type AddOption func(*AddOpts)

type AddOpts struct {
	deps   []string
	labels []string
}

func Dependencies(deps []string) AddOption {
	return func(o *AddOpts) {
		o.deps = deps
	}
}

func Labels(labels []string) AddOption {
	return func(o *AddOpts) {
		o.labels = labels
	}
}

func (g *DAG) Add(node string, opt ...AddOption) bool {
	opts := &AddOpts{}
	for _, o := range opt {
		o(opts)
	}

	g.AddNode(node)

	for _, d := range opts.deps {
		g.Add(d)
	}

	deps := g.AddDependencies(node, opts.deps)

	g.AddLabels(node, opts.labels)

	return deps
}

func (g *DAG) unsafeRemoveEdge(from, to string) {
	delete(g.outputs[from], to)
	g.numInputs[to]--
}

func (g *DAG) RemoveEdge(from, to string) bool {
	if _, ok := g.outputs[from]; !ok {
		return false
	}
	g.unsafeRemoveEdge(from, to)
	return true
}

type NodeInfo struct {
	Id        string
	ParentIds []string
	ChildIds  []string
}

func (n *NodeInfo) String() string {
	return n.Id
}

// Cycle is not a loop :)
// See https://math.stackexchange.com/questions/1490053
type Cycle struct {
	Path []string
}

func (n *Cycle) String() string {
	return strings.Join(n.Path, " -> ")
}

func (g *DAG) Plan() (Topology, error) {
	return g.Sort()
}

type Topology [][]*NodeInfo

func (r Topology) String() string {
	if len(r) == 0 {
		return ""
	}

	res := []string{}

	for _, set := range r {
		ids := []string{}
		for _, n := range set {
			ids = append(ids, n.Id)
		}
		sort.Strings(ids)
		res = append(res, strings.Join(ids, ", "))
	}

	return strings.Join(res, " -> ")
}

func depthFirstPrint(nodes map[string]*NodeInfo, level int, levels map[int]map[string]bool, n *NodeInfo) [][]string {
	lines := [][]string{}

	if len(n.ChildIds) == 0 {
		return [][]string{{n.Id}}
	}

	for _, child := range n.ChildIds {
		if ok := levels[level+1][child]; !ok {
			continue
		}
		childLines := depthFirstPrint(nodes, level+1, levels, nodes[child])

		for i, line := range childLines {
			var header []string
			if i == 0 {
				header = []string{n.Id}
			} else {
				header = []string{""}
			}
			lines = append(lines, append(header, line...))
		}
	}

	return lines
}

type Error struct {
	Cycle *Cycle
}

func (e *Error) Error() string {
	return fmt.Sprintf("cycle detected: %v", e.Cycle)
}

// Sort topologically sorts the nodes while grouping nodes at the same "depth" into a same group
func (g *DAG) Sort() (Topology, error) {
	nodes := map[string]*NodeInfo{}
	current := make([]*NodeInfo, 0, len(g.nodes))

	for _, n := range g.nodes {
		info := &NodeInfo{Id: n}
		nodes[n] = info
		if g.numInputs[n] == 0 {
			current = append(current, info)
		}
	}

	// We sort sets of nodes rather than nodes themselves,
	// so that we know which items can be processed in parallel in the DAG
	// See https://cs.stackexchange.com/questions/2524/getting-parallel-items-in-dependency-resolution
	sortedSets := map[int][]*NodeInfo{}

	next := []*NodeInfo{}

	depth := 0
	for len(current) > 0 {
		var n *NodeInfo
		n, current = current[0], current[1:]
		sortedSets[depth] = append(sortedSets[depth], n)

		ms := make([]string, len(g.outputs[n.Id]))
		i := 0
		for m := range g.outputs[n.Id] {
			ms[i] = m
			i++
		}

		for _, m := range ms {
			g.unsafeRemoveEdge(n.Id, m)

			mm := nodes[m]
			mm.ParentIds = append(mm.ParentIds, n.Id)

			n.ChildIds = append(n.ChildIds, mm.Id)

			if g.numInputs[m] == 0 {
				next = append(next, mm)
			}
		}

		if len(current) == 0 {
			current = next
			next = []*NodeInfo{}
			depth += 1
		}
	}

	invalidNodes := []*NodeInfo{}

	numUnresolvedEdges := 0
	for id, v := range g.numInputs {
		numUnresolvedEdges += v

		if v > 0 {
			invalidNodes = append(invalidNodes, nodes[id])
		}
	}

	if numUnresolvedEdges > 0 {
		var cur string

		// Sort Ids to make the result stable
		sorted := []string{}
		for _, n := range invalidNodes {
			sorted = append(sorted, n.Id)
		}

		sort.Strings(sorted)

		for _, id := range sorted {
			if len(g.outputs[id]) > 0 {
				cur = id
				break
			}
		}
		if cur == "" {
			panic(fmt.Errorf("invalid state: no nodes have remaining edges: nodes=%v", invalidNodes))
		}
		seen := map[string]bool{}
		path := []string{}

		for !seen[cur] {
			seen[cur] = true
			path = append(path, cur)
			for k, _ := range g.outputs[cur] {
				cur = k
				break
			}
		}
		path = append(path, cur)

		r := make(Topology, len(sortedSets))
		for k, v := range sortedSets {
			r[k] = v
		}

		return r, &Error{Cycle: &Cycle{Path: path}}
	}

	r := make(Topology, len(sortedSets))
	for k, v := range sortedSets {
		r[k] = v
	}

	return r, nil
}
