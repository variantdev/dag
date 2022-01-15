package dag

import (
	"testing"
)

type helmReleaseKey struct {
	context   string
	namespace string
	name      string
}

func (a helmReleaseKey) Less(k Key) bool {
	b := k.(helmReleaseKey)

	return a.context < b.context || a.namespace < b.namespace || a.name < b.name
}

func (a helmReleaseKey) String() string {
	return a.name
}

func key(name string) helmReleaseKey {
	return helmReleaseKey{name: name}
}

func TestDAG(t *testing.T) {

	var (
		web  = key("web")
		api1 = key("api1")
		api2 = key("api2")
		db1  = key("db1")
		db2  = key("db2")
		db3  = key("db3")
		net  = key("net")
	)

	g2 := New()
	g2.Add(
		web,
		Dependencies(
			api1,
			api2,
		),
	)
	g2.Add(api1, Dependencies(db1))
	g2.Add(api2, Dependencies(db2))
	g2.Add(db1, Dependencies(net))
	g2.Add(db2, Dependencies(net))
	g2.Add(db3, Dependencies(net))
	g2.Add(net)

	var res Topology
	var err error

	res, err = g2.Plan(Only(api1))
	if err == nil || err.Error() != "\"db1\" depended by \"api1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if ude, ok := err.(*UnhandledDependencyError); ok {
		if n := len(ude.UnhandledDependencies); n != 1 {
			t.Fatalf("unexpected length of unhandled dependencies in error: %v", n)
		}

		ud := ude.UnhandledDependencies[0]

		if ud.Id != db1 {
			t.Fatalf("unexpected id of unhandled dependency: %v", ud.Id)
		}

		if n := len(ud.Dependents); n != 1 {
			t.Fatalf("unexpected number of dependents in unhandled dependency: %v", n)
		}

		d := ud.Dependents[0]

		if d != api1 {
			t.Fatalf("unexpected dependent: %v", d)
		}
	} else {
		t.Fatalf("unexpected type of error: %v(%T)", err, err)
	}

	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(api1), WithoutDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only(api1, db1))
	if err.Error() != "\"net\" depended by \"db1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(api1, db1), WithoutDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only(api1), WithDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "net -> db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only(api1, db1), WithDependencies())
	if err != nil {
		panic(err)
	}
	if expected, actual := "net -> db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only(api1, db1, net))
	if err != nil {
		panic(err)
	}
	if expected, actual := "net -> db1 -> api1", res.String(); actual != expected {
		t.Errorf("unexpected result: expected=%q, got=%q", expected, actual)
	}

	res, err = g2.Plan(Only(api1, db1, net, api2))
	if err.Error() != "\"db2\" depended by \"api2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(api2, db2, net, api1))
	if err.Error() != "\"db1\" depended by \"api1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(api1, net))
	if err.Error() != "\"db1\" depended by \"api1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(api2, net))
	if err.Error() != "\"db2\" depended by \"api2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(db1))
	if err.Error() != "\"net\" depended by \"db1\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(db2))
	if err.Error() != "\"net\" depended by \"db2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(db1, db2))
	if err.Error() != "\"net\" depended by \"db1\" and \"db2\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}

	res, err = g2.Plan(Only(db1, db2, db3))
	if err.Error() != "\"net\" depended by \"db1\", \"db2\", and \"db3\" is not included" {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}
}
