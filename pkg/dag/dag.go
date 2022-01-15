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

func Node(nodes ...Key) Option {
	return func(g *DAG) {
		g.initNodes = nodes
	}
}

func Nodes(nodes []Key) Option {
	return func(g *DAG) {
		g.initNodes = nodes
	}
}

type Key interface {
	Less(r Key) bool
}

type DAG struct {
	cap       int
	initNodes []Key

	nodes []Key
	// a.k.a dependents of the node denoted by the key
	// `outputs["api"]["web"] = true` means api's sole dependent is "web"
	// i.e. "web" depends on "api"
	outputs map[Key]map[Key]bool
	labels  map[Key]map[string]bool
	// a.k.a number of dependenciesthat the node denoted by the key has.
	// `numInputs["web"] = 2` means "web" has 2 dependencies.
	numInputs map[Key]int
}

func (g *DAG) AddNode(key Key) bool {
	if _, ok := g.numInputs[key]; ok {
		return false
	}

	g.nodes = append(g.nodes, key)

	if _, ok := g.outputs[key]; !ok {
		g.outputs[key] = make(map[Key]bool)
	}

	if _, ok := g.numInputs[key]; ok {
		g.numInputs[key] = 0
	}

	return true
}

func New(opt ...Option) *DAG {
	g := &DAG{
		numInputs: make(map[Key]int),
		outputs:   make(map[Key]map[Key]bool),
		labels:    make(map[Key]map[string]bool),
	}

	for _, o := range opt {
		o(g)
	}

	g.nodes = make([]Key, 0, g.cap)

	g.AddNodes(g.initNodes...)

	return g
}

func (g *DAG) AddNodes(names ...Key) bool {
	for _, name := range names {
		if ok := g.AddNode(name); !ok {
			return false
		}
	}
	return true
}

func (g *DAG) AddEdge(from, to Key) bool {
	m, ok := g.outputs[from]
	if !ok {
		m = map[Key]bool{}
		g.outputs[from] = m
	}

	m[to] = true
	g.numInputs[to]++

	return true
}

func (g *DAG) AddDependency(sub Key, dependencies ...Key) bool {
	for _, d := range dependencies {
		if r := g.AddEdge(d, sub); !r {
			return false
		}
	}
	return true
}

func (g *DAG) AddDependencies(sub Key, dependencies []Key) bool {
	return g.AddDependency(sub, dependencies...)
}

func (g *DAG) AddLabel(sub Key, labels ...string) {
	for _, d := range labels {
		m, ok := g.labels[sub]
		if !ok {
			m = map[string]bool{}
			g.labels[sub] = m
		}
		m[d] = true
	}
}

func (g *DAG) AddLabels(sub Key, labels []string) {
	g.AddLabel(sub, labels...)
}

type AddOption func(*AddOpts)

type AddOpts struct {
	deps   []Key
	labels []string
}

func Dependencies(deps ...Key) AddOption {
	return func(o *AddOpts) {
		o.deps = deps
	}
}

func Labels(labels []string) AddOption {
	return func(o *AddOpts) {
		o.labels = labels
	}
}

func (g *DAG) Add(node Key, opt ...AddOption) bool {
	opts := &AddOpts{}
	for _, o := range opt {
		o(opts)
	}

	g.AddNode(node)

	deps := g.AddDependencies(node, opts.deps)

	g.AddLabels(node, opts.labels)

	return deps
}

func (g *DAG) unsafeRemoveEdge(from, to Key) {
	delete(g.outputs[from], to)
	g.numInputs[to]--
}

func (g *DAG) RemoveEdge(from, to Key) bool {
	if _, ok := g.outputs[from]; !ok {
		return false
	}
	g.unsafeRemoveEdge(from, to)
	return true
}

type NodeInfo struct {
	Id        Key
	ParentIds []Key
	ChildIds  []Key
}

func (n *NodeInfo) String() string {
	return fmt.Sprintf("%s", n.Id)
}

// Cycle is not a loop :)
// See https://math.stackexchange.com/questions/1490053
type Cycle struct {
	Path []Key
}

func (n *Cycle) String() string {
	var path []string
	for _, p := range n.Path {
		path = append(path, fmt.Sprintf("%s", p))
	}

	return strings.Join(path, " -> ")
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
		ids := []Key{}
		for _, n := range set {
			ids = append(ids, n.Id)
		}
		sort.Slice(ids, func(i, j int) bool {
			return ids[i].Less(ids[j])
		})
		res = append(res, strings.Join(KeysToStringSlice(ids), ", "))
	}

	return strings.Join(res, " -> ")
}

func sprintKey(k Key) string {
	return fmt.Sprintf("%s", k)
}

func KeysToStringSlice(ks []Key) []string {
	var ss []string

	for _, k := range ks {
		ss = append(ss, sprintKey(k))
	}

	return ss
}

func depthFirstPrint(nodes map[Key]*NodeInfo, level int, levels map[int]map[Key]bool, n *NodeInfo) [][]string {
	lines := [][]string{}

	if len(n.ChildIds) == 0 {
		return [][]string{{sprintKey(n.Id)}}
	}

	for _, child := range n.ChildIds {
		if ok := levels[level+1][child]; !ok {
			continue
		}
		childLines := depthFirstPrint(nodes, level+1, levels, nodes[child])

		for i, line := range childLines {
			var header []string
			if i == 0 {
				header = []string{sprintKey(n.Id)}
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
	UndefinedNode Key
	Dependents    []Key
}

func (e *UndefinedDependencyError) Error() string {
	return fmt.Sprintf("undefined node %q is depended by node(s): %s", e.UndefinedNode, strings.Join(KeysToStringSlice(e.Dependents), ", "))
}

type UnhandledDependencyError struct {
	UnhandledDependencies []UnhandledDependency
}

type UnhandledDependency struct {
	Id         Key
	Dependents []Key
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
	Only []Key

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

func Only(nodes ...Key) SortOption {
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

	var only map[Key]struct{}

	if len(options.Only) > 0 {
		only = map[Key]struct{}{}

		for _, o := range options.Only {
			only[o] = struct{}{}
		}
	}

	numInputs := map[Key]int{}
	for k, v := range g.numInputs {
		numInputs[k] = v
	}

	outputs := map[Key]map[Key]bool{}
	for k, v := range g.outputs {
		outputs[k] = map[Key]bool{}
		for k2, v2 := range v {
			outputs[k][k2] = v2
		}
	}

	withDeps := options.WithDependencies
	withoutDeps := options.WithoutDependencies

	nodes := map[Key]*NodeInfo{}
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
			var dependentsNames []Key
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

		ms := make([]Key, len(outputs[n.Id]))
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
		var cur Key

		// Sort Ids to make the result stable
		sorted := []Key{}
		for _, n := range invalidNodes {
			sorted = append(sorted, n.Id)
		}

		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Less(sorted[j])
		})

		for _, id := range sorted {
			if len(outputs[id]) > 0 {
				cur = id
				break
			}
		}
		if cur == nil {
			panic(fmt.Errorf("invalid state: no nodes have remaining edges: nodes=%v", invalidNodes))
		}
		seen := map[Key]bool{}
		path := []Key{}

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
			var dependents []Key

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
					sort.Slice(dependents, func(i, j int) bool {
						return dependents[i].Less(dependents[j])
					})

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
			return included[i].Id.Less(included[j].Id)
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
