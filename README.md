# dag

Topologically sortable DAG implementation for Go with support for parallel items.

## Examples

This package provides two sub-packages.

`strdag` accepts `string` as node key. `dag` is a more general variant that accepts `Key` as node key.

- [strdag package](#strdag)
- [dag package](#dag)

### strdag package

Planning the order of application deployments, so that all apps are ensured to be deployed after their dependencies are ready:

```golang
import (
    dag "github.com/variantdev/dag/pkg/strdag"
)

g := dag.New(dag.Nodes([]string{"web", "api", "db", "cache", "mesh", "net"}))
g.AddDependencies("web", []string{"api", "cache", "net"})
g.AddDependencies("api", []string{"db", "cache", "net"})
g.AddDependencies("db", []string{"net"})
g.AddDependencies("mesh", []string{"net"})

res, err := g.Plan()
if err != nil {
    panic(err)
}

res.String()
// => "cache, net -> db, mesh -> api -> web"

// Writes Graphviz' Dot representation of the graph
//
// Render it with:
//   brew install graphviz
//   pbpaste | dot -Tpng | imgcat
// Or:
//   whalebrew install tsub/graph-easy
//   pbpaste | graph-easy
res.WriteDotTo(os.Stdout)
```

### Scoping the DAG to only include a subset of nodes

You can pass some `Only(nodeName)` argument to the `Plan` function to scope the DAG.

```golang
res, err := g.Plan(dag.Only("net"))
if err != nil {
    panic(err)
}

res.String()
// => "net"
```

Beware that the `Plan` function requires you to explicitly specify how it should handle the missing dependency.

In the previous example, `db` and `mesh` depends on `net` so trying to include only `db` and `mesh` returns an error basically saying you need to explicitly specify how to handle `net`:

```golang
res, err := g.Plan(dag.Only("db", "mesh"))
if err != nil {
    // => "net" depended by "db" and "mesh" is not included
    panic(err)
}
```

To skip `net`, pass `WithoutDependencies`:

```golang
res, err := g.Plan(dag.Only("db", "mesh"), dag.WithoutDependencies())
res.String()
// => "db, mesh"
```

To implicitly include `net` into the DAG, pass `WithDependencies` as well:

```golang
res, err := g.Plan(dag.Only("db", "mesh"), dag.WithDependencies())

res.String()
// => "net -> db, mesh"
```

It can be obvious, but you can even expilcitly include `net` into the DAG, by adding it to `Only`...:

```
res, _ := g.Plan(dag.Only("db", "mesh", "net"))

res.String()
// => "net -> db, mesh"
```

Also note that you can type-assert the error object to `*UnhandledDependencyError` to grab more detailed information about the error:

```
if ude, ok := err.(*UnhandledDependencyError); ok {
    ude.UnhandledDependencies[0].Id
    // => "net"
    ude.UnhandledDependencies[0].Dependents
    // => {db mesh}
}
```

### dag package

`dag` package works almost the same as `strdag` package explained above.

The only difference is that it accepts anything that implements the `dag.Key` interface.

I believe you'll usually find it more easy to use `strdag`. But you sometimes need to use `dag` when you want to use any struct as key.
