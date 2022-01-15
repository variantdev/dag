package strdag

import (
	"fmt"
	"io"

	"github.com/variantdev/dag/pkg/dag"
)

type StringKey string

func (s StringKey) Less(r dag.Key) bool {
	sk, ok := r.(StringKey)
	if !ok {
		panic(fmt.Sprintf("unexpected type of Key %T: %v", r, r))
	}

	return string(s) < string(sk)
}

func stringsToKeys(ss []string) []dag.Key {
	var ks []dag.Key

	for _, s := range ss {
		ks = append(ks, StringKey(s))
	}

	return ks
}

type DAG struct {
	d *dag.DAG
}

type Option = dag.Option
type SortOption = dag.SortOption

type UnhandledDependencyError struct {
	*dag.UnhandledDependencyError

	UnhandledDependencies []UnhandledDependency
}

type UnhandledDependency struct {
	Id         string
	Dependents []string
}

func (e *UnhandledDependencyError) Error() string {
	return e.UnhandledDependencyError.Error()
}

// Option

var Capacity = dag.Capacity

// AddOption

var Labels = dag.Labels

// SortOption

var WithDependencies = dag.WithDependencies
var WithoutDependencies = dag.WithoutDependencies

func Nodes(ids []string) Option {
	return dag.Nodes(stringsToKeys(ids))
}

func Dependencies(ids []string) dag.AddOption {
	return dag.Dependencies(stringsToKeys(ids)...)
}

func Only(ids ...string) dag.SortOption {
	return dag.Only(stringsToKeys(ids)...)
}

func New(opt ...Option) *DAG {
	d := dag.New(opt...)

	return &DAG{d: d}
}

func (d *DAG) Add(id string, opts ...dag.AddOption) {
	d.d.Add(StringKey(id), opts...)
}

func (d *DAG) AddNodes(ids ...string) {
	d.d.AddNodes(stringsToKeys(ids)...)
}

func (d *DAG) AddEdge(from, to string) {
	d.d.AddEdge(StringKey(from), StringKey(to))
}

func (d *DAG) AddDependencies(id string, deps []string) {
	d.d.AddDependencies(StringKey(id), stringsToKeys(deps))
}

type Topology [][]*NodeInfo

func (r Topology) String() string {
	var t dag.Topology

	for _, group := range r {
		var newGroup []*dag.NodeInfo

		for _, e := range group {
			newGroup = append(newGroup, &dag.NodeInfo{
				Id:        StringKey(e.Id),
				ParentIds: stringsToKeys(e.ParentIds),
				ChildIds:  stringsToKeys(e.ChildIds),
			})
		}

		t = append(t, newGroup)
	}

	return t.String()
}

type NodeInfo struct {
	Id        string
	ParentIds []string
	ChildIds  []string
}

func (n *NodeInfo) String() string {
	return fmt.Sprintf("%s", n.Id)
}

func (d *DAG) Sort(opts ...SortOption) (Topology, error) {
	return transformPlanResAndErr(d.d.Sort(opts...))
}

func (d *DAG) Plan(opts ...SortOption) (Topology, error) {
	return transformPlanResAndErr(d.d.Plan(opts...))
}

func (d *DAG) WriteDotTo(w io.Writer) error {
	return d.d.WriteDotTo(w)
}

func transformPlanResAndErr(t dag.Topology, err error) (Topology, error) {
	if err != nil {
		ude, ok := err.(*dag.UnhandledDependencyError)
		if ok {
			var uds []UnhandledDependency

			for _, ud := range ude.UnhandledDependencies {
				var deps []string

				for _, d := range ud.Dependents {
					deps = append(deps, fmt.Sprintf("%s", d))
				}

				uds = append(uds, UnhandledDependency{
					Id:         fmt.Sprintf("%s", ud.Id),
					Dependents: deps,
				})
			}

			err = &UnhandledDependencyError{
				UnhandledDependencyError: ude,
				UnhandledDependencies:    uds,
			}
		}
	}

	var transformed Topology

	for _, group := range t {
		var infoTransformed []*NodeInfo

		for _, info := range group {
			infoTransformed = append(infoTransformed, &NodeInfo{
				Id:        fmt.Sprintf("%s", info.Id),
				ParentIds: dag.KeysToStringSlice(info.ParentIds),
				ChildIds:  dag.KeysToStringSlice(info.ChildIds),
			})
		}

		transformed = append(transformed, infoTransformed)
	}

	return transformed, err
}
