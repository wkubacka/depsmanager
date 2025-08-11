package diff

import (
	"depsmanager"
	"math"
)

func DiffDependencies(a, b []depsmanager.Dependency) (onlyA, onlyB []depsmanager.Dependency) {
	inB := make(map[string]depsmanager.Dependency, len(b))
	for _, d := range b {
		inB[d.Name] = d
	}

	for _, d := range a {
		if x, ok := inB[d.Name]; !ok || !equalDep(d, x) {
			onlyA = append(onlyA, d)
		}
	}

	inA := make(map[string]depsmanager.Dependency, len(a))
	for _, d := range a {
		inA[d.Name] = d
	}

	for _, d := range b {
		if x, ok := inA[d.Name]; !ok || !equalDep(d, x) {
			onlyB = append(onlyB, d)
		}
	}

	return onlyA, onlyB
}

func equalDep(a, b depsmanager.Dependency) bool {
	return a.Name == b.Name &&
		floatEq(a.Score, b.Score) &&
		a.UpdatedAt == b.UpdatedAt
}

func floatEq(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) <= eps
}
