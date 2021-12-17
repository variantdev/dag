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

	nodes []string
	// a.k.a dependents of the node denoted by the key
	// `outputs["api"]["web"] = true` means api's sole dependent is "web"
	// i.e. "web" depends on "api"
	outputs map[string]map[string]bool
	labels  map[string]map[string]bool
	// a.k.a number of dependenciesthat the node denoted by the key has.
	// `numInputs["web"] = 2` means "web" has 2 dependencies.
	numInputs map[string]int
}

func (g *DAG) AddNode(name string) bool {
	if _, ok := g.numInputs[name]; ok {
		return false
	}

	g.nodes = append(g.nodes, name)

	if _, ok := g.outputs[name]; !ok {
		g.outputs[name] = make(map[string]bool)
	}

	if _, ok := g.numInputs[name]; ok {
		g.numInputs[name] = 0
	}

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
		m = map[string]bool{}
		g.outputs[from] = m
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

func (g *DAG) Plan(opts ...SortOption) (Topology, error) {
	return g.Sort(opts...)
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

type UndefinedDependencyError struct {
	UndefinedNode string
	Dependents    []string
}

func (e *UndefinedDependencyError) Error() string {
	return fmt.Sprintf("undefined node %q is depended by node(s): %s", e.UndefinedNode, strings.Join(e.Dependents, ", "))
}

type UnhandledDependencyError struct {
	UnhandledDependencies []UnhandledDependency
}

type UnhandledDependency struct {
	Id         string
	Dependents []string
}

func (e *UnhandledDependencyError) Error() string {
	ud := e.UnhandledDependencies[0]

	dependents := make([]string, len(ud.Dependents))

	for i := 0; i < len(dependents); i++ {
		dependents[i] = fmt.Sprintf("%q", ud.Dependents[i])
	}

	var ds string

	if len(dependents) < 3 {
		ds = strings.Join(dependents, " and ")
	} else {
		ds = strings.Join(dependents[:len(dependents)-1], ", ")
		ds += ", and " + dependents[len(dependents)-1]
	}

	return fmt.Sprintf("%q depended by %s is not included", ud.Id, ds)
}

type SortOptions struct {
	Only []string

	WithDependencies bool

	WithoutDependencies bool
}

func (so SortOptions) ApplySortOptions(dst *SortOptions) {
	*dst = so
}

type SortOption interface {
	ApplySortOptions(*SortOptions)
}

type sortOptionFunc struct {
	f func(so *SortOptions)
}

func (sof *sortOptionFunc) ApplySortOptions(so *SortOptions) {
	sof.f(so)
}

func SortOptionFunc(f func(so *SortOptions)) SortOption {
	return &sortOptionFunc{
		f: f,
	}
}

func Only(nodes ...string) SortOption {
	return SortOptionFunc(func(so *SortOptions) {
		so.Only = append(so.Only, nodes...)
	})
}

func WithDependencies() SortOption {
	return SortOptionFunc(func(so *SortOptions) {
		so.WithDependencies = true
	})
}

func WithoutDependencies() SortOption {
	return SortOptionFunc(func(so *SortOptions) {
		so.WithoutDependencies = true
	})
}

// Sort topologically sorts the nodes while grouping nodes at the same "depth" into a same group
func (g *DAG) Sort(opts ...SortOption) (Topology, error) {
	var options SortOptions

	for _, o := range opts {
		o.ApplySortOptions(&options)
	}

	var only map[string]struct{}

	if len(options.Only) > 0 {
		only = map[string]struct{}{}

		for _, o := range options.Only {
			only[o] = struct{}{}
		}
	}

	numInputs := map[string]int{}
	for k, v := range g.numInputs {
		numInputs[k] = v
	}

	outputs := map[string]map[string]bool{}
	for k, v := range g.outputs {
		outputs[k] = map[string]bool{}
		for k2, v2 := range v {
			outputs[k][k2] = v2
		}
	}

	withDeps := options.WithDependencies
	withoutDeps := options.WithoutDependencies

	nodes := map[string]*NodeInfo{}
	current := make([]*NodeInfo, 0, len(g.nodes))

	for _, n := range g.nodes {
		info := &NodeInfo{Id: n}
		nodes[n] = info
		if numInputs[n] == 0 {
			current = append(current, info)
		}
	}

	for dep, dependents := range outputs {
		if _, ok := nodes[dep]; !ok {
			var dependentsNames []string
			for d := range dependents {
				dependentsNames = append(dependentsNames, d)
			}
			return nil, &UndefinedDependencyError{
				UndefinedNode: dep,
				Dependents:    dependentsNames,
			}
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

		ms := make([]string, len(outputs[n.Id]))
		i := 0
		for m := range outputs[n.Id] {
			ms[i] = m
			i++
		}

		for _, m := range ms {
			// unsafeRemoveEdge
			from, to := n.Id, m
			delete(outputs[from], to)
			numInputs[to]--

			mm := nodes[m]
			mm.ParentIds = append(mm.ParentIds, n.Id)

			n.ChildIds = append(n.ChildIds, mm.Id)

			if numInputs[m] == 0 {
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
	for id, v := range numInputs {
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
			if len(outputs[id]) > 0 {
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
			for k, _ := range outputs[cur] {
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

	for k := len(sortedSets) - 1; k >= 0; k-- {
		v := sortedSets[k]

		var included []*NodeInfo

		for i := range v {
			node := v[i]

			if only == nil {
				included = append(included, node)
				continue
			}

			if _, ok := only[node.Id]; ok {
				included = append(included, node)
				continue
			}

			if withoutDeps {
				continue
			}

			var depended bool
			var dependents []string

			allDependents := g.outputs[node.Id]
			for target := range only {
				if allDependents[target] {
					// This node is depended by one of the selected nodes
					depended = true
					dependents = append(dependents, target)
				}
			}

			if depended {
				// The user has not opted-in to automatically include this node as depended by the one of the selected nodes
				if !withDeps {
					sort.Strings(dependents)

					return nil, &UnhandledDependencyError{
						UnhandledDependencies: []UnhandledDependency{
							{
								Id:         node.Id,
								Dependents: dependents,
							},
						},
					}
				}

				// The user has opted-in to automatically include this node as the dependency of one of the selected nodes

				// To include any transitive dependencies of the this node into the dag,
				// we treat this node as included in the selected nodes list.
				only[node.Id] = struct{}{}
				included = append(included, node)
				continue
			}
		}

		if len(included) == 0 {
			continue
		}

		sort.Slice(included, func(i, j int) bool {
			return included[i].Id < included[j].Id
		})

		r[k] = included
	}

	res := [][]*NodeInfo{}

	for _, ns := range r {
		if len(ns) > 0 {
			res = append(res, ns)
		}
	}

	return res, nil
}
