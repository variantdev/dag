# dag

Topologically sortable DAG implementation for Go with support for parallel items.

## Examples

Planning the order of application deployments, so that all apps are ensured to be deployed after their dependencies are ready:

```golang
g := dag.New(dag.Nodes([]string{"web", "api", "db", "cache", "mesh", "net"}))
g.AddDependencies("web", []string{"api", "cache", "net"})
g.AddDependencies("api", []string{"db", "cache", "net"})
g.AddDependencies("db", []string{"net"})
g.AddDependencies("mesh", []string{"net"})

res, err := g.Plan()
if err != nil {
    panic(err)
}

g.String()
// => "cache, net -> db, mesh -> api -> web"

// Writes Graphviz' Dot representation of the graph
//
// Render it with:
//   brew install graphviz
//   pbpaste | dot -Tpng | imgcat
// Or:
//   whalebrew install tsub/graph-easy
//   pbpaste | graph-easy
g.WriteDotTo(os.Stdout)
```
