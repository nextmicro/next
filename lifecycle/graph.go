package lifecycle

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// CyclicDependencyError 表示组件依赖中检测到环。
type CyclicDependencyError struct {
	Cycle []string
}

func (e *CyclicDependencyError) Error() string {
	return "lifecycle: cyclic dependency detected: " + strings.Join(e.Cycle, " -> ")
}

// MissingDependencyError 表示某 Component 声明的依赖未注册到 Lifecycle。
type MissingDependencyError struct {
	Owner   string
	Missing reflect.Type
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("lifecycle: component %q depends on unregistered type %s", e.Owner, e.Missing.String())
}

// topoSort 对给定 Component 列表按依赖拓扑排序并分层。
// 同层 Component 彼此无依赖关系，可并行启动。
//
// 算法：Kahn's algorithm 的分层变体。
//   - 每轮取出当前入度为 0 的节点作为新层
//   - 移除这些节点对应的出边，重复直到全部输出
//   - 若仍有节点剩余则存在环
//
// 返回值：[][]Component，外层为层，内层为同层 Component。
func topoSort(comps []Component) ([][]Component, error) {
	if len(comps) == 0 {
		return nil, nil
	}

	byType := make(map[reflect.Type]Component, len(comps))
	for _, c := range comps {
		t := componentType(c)
		if _, exists := byType[t]; exists {
			return nil, fmt.Errorf("lifecycle: duplicate component type %s", t.String())
		}
		byType[t] = c
	}

	// 构建邻接表与入度表
	inDegree := make(map[reflect.Type]int, len(comps))
	edges := make(map[reflect.Type][]reflect.Type) // dep -> [dependents]
	for _, c := range comps {
		t := componentType(c)
		inDegree[t] = 0
	}
	for _, c := range comps {
		t := componentType(c)
		dep, ok := c.(DependsOn)
		if !ok {
			continue
		}
		for _, depType := range dep.Deps() {
			if _, exists := byType[depType]; !exists {
				return nil, &MissingDependencyError{Owner: c.Name(), Missing: depType}
			}
			edges[depType] = append(edges[depType], t)
			inDegree[t]++
		}
	}

	// 分层 Kahn
	var layers [][]Component
	remaining := len(comps)

	for remaining > 0 {
		// 找出当前入度为 0 的节点
		var ready []reflect.Type
		for t, deg := range inDegree {
			if deg == 0 {
				ready = append(ready, t)
			}
		}
		if len(ready) == 0 {
			// 还有节点但全部入度 > 0 → 存在环
			return nil, &CyclicDependencyError{Cycle: extractCycle(byType, inDegree, edges)}
		}

		// 同层内按 Name 排序，保证启动顺序稳定可预测
		sort.Slice(ready, func(i, j int) bool {
			return byType[ready[i]].Name() < byType[ready[j]].Name()
		})

		layer := make([]Component, 0, len(ready))
		for _, t := range ready {
			layer = append(layer, byType[t])
			delete(inDegree, t)
			for _, dependent := range edges[t] {
				inDegree[dependent]--
			}
		}
		layers = append(layers, layer)
		remaining -= len(layer)
	}

	return layers, nil
}

// extractCycle 在拓扑排序失败时尽力提取一个环以便错误信息可读。
// 实现：在剩余节点上做 DFS，遇到已在当前路径上的节点即返回该环。
func extractCycle(
	byType map[reflect.Type]Component,
	remaining map[reflect.Type]int,
	edges map[reflect.Type][]reflect.Type,
) []string {
	visited := make(map[reflect.Type]bool)
	onPath := make(map[reflect.Type]bool)
	var path []reflect.Type
	var cycle []string

	var dfs func(t reflect.Type) bool
	dfs = func(t reflect.Type) bool {
		if onPath[t] {
			// 找到环：截取 path 从首次出现 t 处开始
			start := 0
			for i, p := range path {
				if p == t {
					start = i
					break
				}
			}
			for _, p := range path[start:] {
				cycle = append(cycle, byType[p].Name())
			}
			cycle = append(cycle, byType[t].Name())
			return true
		}
		if visited[t] {
			return false
		}
		visited[t] = true
		onPath[t] = true
		path = append(path, t)
		for _, next := range edges[t] {
			if _, stillIn := remaining[next]; !stillIn {
				continue
			}
			if dfs(next) {
				return true
			}
		}
		onPath[t] = false
		path = path[:len(path)-1]
		return false
	}

	for t := range remaining {
		if dfs(t) {
			return cycle
		}
	}
	// 兜底
	names := make([]string, 0, len(remaining))
	for t := range remaining {
		names = append(names, byType[t].Name())
	}
	sort.Strings(names)
	return names
}

// collectDeps 收集所有 Component 的依赖列表，键为 Component.Name()。
// 供 Container.Validate 调用。
func collectDeps(comps []Component) map[string][]reflect.Type {
	out := make(map[string][]reflect.Type, len(comps))
	for _, c := range comps {
		if dep, ok := c.(DependsOn); ok {
			out[c.Name()] = dep.Deps()
		}
	}
	return out
}
