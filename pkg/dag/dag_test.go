package dag

import (
	"bytes"
	"fmt"
	"log"
	"testing"
)

func TestDAG_GraphAPI(t *testing.T) {
	g1 := New(Capacity(8))
	g1.AddNodes("2", "3", "5", "7", "8", "9", "10", "11")

	g1.AddEdge("7", "8")
	g1.AddEdge("7", "11")

	g1.AddEdge("5", "11")
	//g1.AddEdge("5", "8")

	g1.AddEdge("3", "8")
	g1.AddEdge("3", "10")

	g1.AddEdge("11", "2")
	g1.AddEdge("11", "9")
	g1.AddEdge("11", "10")

	g1.AddEdge("8", "9")

	result, err := g1.Sort()
	if err != nil {
		panic(err)
	}

	expected := "3, 5, 7 -> 11, 8 -> 10, 2, 9"
	actual := result.String()
	if actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}
}

func TestDAG_DagAPI(t *testing.T) {
	g2 := New(
		Nodes([]string{"web", "api", "db", "cache", "mesh", "net"}),
	)
	g2.AddDependencies(
		"web",
		[]string{
			"api",
			"cache",
			"net",
		},
	)
	g2.AddDependencies("api", []string{"db", "cache", "net"})
	g2.AddDependencies("db", []string{"net"})
	g2.AddDependencies("mesh", []string{"net"})

	res, err := g2.Plan()
	if err != nil {
		panic(err)
	}

	expected := "cache, net -> db, mesh -> api -> web"
	actual := res.String()
	if actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	groups := [][]string{
		{"cache", "net"},
		{"db", "mesh"},
		{"api"},
		{"web"},
	}

	for i, g := range groups {
		for j, expected := range g {
			actual := res[i][j].Id
			if actual != expected {
				t.Errorf("unexpected id at %d, %d: expeted=%q, got=%q", i, j, expected, actual)
			}
		}
	}
}

func TestDAG_DagCleanAPI(t *testing.T) {
	g2 := New()
	g2.Add(
		"web",
		Dependencies([]string{
			"api",
			"cache",
			"net",
		}),
	)
	g2.Add("api", Dependencies([]string{"db", "cache", "net"}))
	g2.Add("db", Dependencies([]string{"net"}))
	g2.Add("mesh", Dependencies([]string{"net"}))
	g2.Add("net")
	g2.Add("cache")

	res, err := g2.Plan()
	if err != nil {
		panic(err)
	}

	expected := "cache, net -> db, mesh -> api -> web"
	actual := res.String()
	if actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}
}

func TestDAG_DagCleanAPI_PlanWithOptions(t *testing.T) {
	g2 := New()
	g2.Add(
		"web",
		Dependencies([]string{
			"api1",
			"api2",
		}),
	)
	g2.Add("api1", Dependencies([]string{"db1"}))
	g2.Add("api2", Dependencies([]string{"db2"}))
	g2.Add("db1", Dependencies([]string{"net"}))
	g2.Add("db2", Dependencies([]string{"net"}))
	g2.Add("db3", Dependencies([]string{"net"}))
	g2.Add("net")

	var res Topology
	var err error

	res, err = g2.Plan(Only("api1"))
	if err == nil || err.Error() != "\"db1\" depended by \"api1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if ude, ok := err.(*UnhandledDependencyError); ok {
		if n := len(ude.UnhandledDependencies); n != 1 {
			t.Fatalf("unexpected length of unhandled dependencies in error: %v", n)
		}

		ud := ude.UnhandledDependencies[0]

		if ud.Id != "db1" {
			t.Fatalf("unexpected id of unhandled dependency: %v", ud.Id)
		}

		if n := len(ud.Dependents); n != 1 {
			t.Fatalf("unexpected number of dependents in unhandled dependency: %v", n)
		}

		d := ud.Dependents[0]

		if d != "api1" {
			t.Fatalf("unexpected dependent: %v", d)
		}
	} else {
		t.Fatalf("unexpected type of error: %v(%T)", err, err)
	}

	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("api1"), WithoutDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only("api1", "db1"))
	if err.Error() != "\"net\" depended by \"db1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("api1", "db1"), WithoutDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only("api1"), WithDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "net -> db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only("api1", "db1"), WithDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "net -> db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only("api1", "db1", "net"))
	if err != nil {
		panic(err)
	}
	if expected, actual := "net -> db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only("api1", "db1", "net", "api2"))
	if err.Error() != "\"db2\" depended by \"api2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("api2", "db2", "net", "api1"))
	if err.Error() != "\"db1\" depended by \"api1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("api1", "net"))
	if err.Error() != "\"db1\" depended by \"api1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("api2", "net"))
	if err.Error() != "\"db2\" depended by \"api2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("db1"))
	if err.Error() != "\"net\" depended by \"db1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("db2"))
	if err.Error() != "\"net\" depended by \"db2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("db1", "db2"))
	if err.Error() != "\"net\" depended by \"db1\" and \"db2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only("db1", "db2", "db3"))
	if err.Error() != "\"net\" depended by \"db1\", \"db2\", and \"db3\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}
}

func TestDAG_Dag_Dot(t *testing.T) {
	g2 := New()
	g2.Add(
		"release/web",
		Dependencies([]string{
			"release/api",
			"release/cache",
			"release/net",
		}),
		Labels([]string{
			"a",
			"b",
			"c",
			"d",
			"e",
		}),
	)
	g2.Add("release/api", Dependencies([]string{"release/db", "release/cache", "release/net"}), Labels([]string{"tier:api"}))
	g2.Add("release/db", Dependencies([]string{"release/net"}), Labels([]string{"tier:db"}))
	g2.Add("release/mesh", Dependencies([]string{"release/net"}), Labels([]string{"tier:net"}))
	g2.Add("release/net")
	g2.Add("release/cache")

	w := &bytes.Buffer{}

	err := g2.WriteDotTo(w)
	if err != nil {
		panic(err)
	}

	{
		actual := w.String()
		// Try it with:
		//   brew install graphviz
		//   pbpaste | dot -Tpng | imgcat
		expected := `digraph DAG {
rankdir="LR"
"release/api" [shape=record, label="{release/api|{tier:api}}"]
"release/cache" [shape=record, label="{release/cache}"]
"release/db" [shape=record, label="{release/db|{tier:db}}"]
"release/mesh" [shape=record, label="{release/mesh|{tier:net}}"]
"release/net" [shape=record, label="{release/net}"]
"release/web" [shape=record, label="{release/web|{a|b|c|d|e}}"]
"release/api" -> "release/web"
"release/cache" -> "release/api"
"release/cache" -> "release/web"
"release/db" -> "release/api"
"release/net" -> "release/api"
"release/net" -> "release/db"
"release/net" -> "release/mesh"
"release/net" -> "release/web"
}
`
		if actual != expected {
			t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
		}
	}
}

func TestDAG_DAGAPICycle(t *testing.T) {
	g2 := New(Nodes([]string{"a", "b", "c"}))
	g2.AddDependencies("a", []string{"b"})
	g2.AddDependencies("b", []string{"c"})
	g2.AddDependencies("c", []string{"a"})

	res, err := g2.Plan()
	if err == nil {
		log.Fatalf("expected error not occuered: %v", res)
	}

	expected := "cycle detected: a -> c -> b -> a"
	actual := err.Error()

	if actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}
}

func TestDAG_GraphAPICycle(t *testing.T) {
	g2 := New(Nodes([]string{"a", "b", "c"}))
	g2.AddEdge("b", "a")
	g2.AddEdge("a", "c")
	g2.AddEdge("c", "b")

	res, err := g2.Sort()
	if err == nil {
		log.Fatalf("expected error not occuered: %v", res)
	}

	expected := "cycle detected: a -> c -> b -> a"
	actual := err.Error()

	if actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}
}

func TestDAG_UndefinedDependency_MaybeTypo(t *testing.T) {
	g2 := New()
	g2.Add(
		"web",
		Dependencies([]string{
			"ok",
			"ng",
		}),
	)
	g2.Add(
		"ok",
	)

	_, err := g2.Plan()

	if err == nil {
		t.Fatalf("expected error didnt occur")
	}

	if fmt.Sprintf("%v", err) != `undefined node "ng" is depended by node(s): web` {
		t.Fatalf("%v", err)
	}
}
