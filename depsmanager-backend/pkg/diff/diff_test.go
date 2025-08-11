package diff

import (
	"depsmanager"
	"reflect"
	"testing"
)

func TestDiffDependencies(t *testing.T) {
	type args struct {
		a []depsmanager.Dependency
		b []depsmanager.Dependency
	}
	tests := []struct {
		name      string
		args      args
		wantOnlyA []depsmanager.Dependency
		wantOnlyB []depsmanager.Dependency
	}{
		{
			name: "identical - empty diffs",
			args: args{
				a: []depsmanager.Dependency{
					{Name: "a", Score: 1.0, UpdatedAt: 10},
					{Name: "b", Score: 2.0, UpdatedAt: 20},
				},
				b: []depsmanager.Dependency{
					{Name: "a", Score: 1.0, UpdatedAt: 10},
					{Name: "b", Score: 2.0, UpdatedAt: 20},
				},
			},
			wantOnlyA: nil,
			wantOnlyB: nil,
		},
		{
			name: "order ignored - empty diffs",
			args: args{
				a: []depsmanager.Dependency{
					{Name: "a", Score: 1.0, UpdatedAt: 10},
					{Name: "b", Score: 2.0, UpdatedAt: 20},
					{Name: "c", Score: 3.0, UpdatedAt: 30},
				},
				b: []depsmanager.Dependency{
					{Name: "c", Score: 3.0, UpdatedAt: 30},
					{Name: "a", Score: 1.0, UpdatedAt: 10},
					{Name: "b", Score: 2.0, UpdatedAt: 20},
				},
			},
			wantOnlyA: nil,
			wantOnlyB: nil,
		},
		{
			name: "no overlap at all - everything ends up in onlyA/onlyB",
			args: args{
				a: []depsmanager.Dependency{
					{Name: "a1", Score: 1.1, UpdatedAt: 11},
					{Name: "a2", Score: 1.2, UpdatedAt: 12},
				},
				b: []depsmanager.Dependency{
					{Name: "b1", Score: 2.1, UpdatedAt: 21},
				},
			},
			wantOnlyA: []depsmanager.Dependency{
				{Name: "a1", Score: 1.1, UpdatedAt: 11},
				{Name: "a2", Score: 1.2, UpdatedAt: 12},
			},
			wantOnlyB: []depsmanager.Dependency{
				{Name: "b1", Score: 2.1, UpdatedAt: 21},
			},
		},
		{
			name: "same name, score within eps - equal, no diffs",
			args: args{
				a: []depsmanager.Dependency{
					{Name: "x", Score: 1.0000000005, UpdatedAt: 100},
				},
				b: []depsmanager.Dependency{
					{Name: "x", Score: 1.0000000004, UpdatedAt: 100},
				},
			},
			wantOnlyA: nil,
			wantOnlyB: nil,
		},
		{
			name: "same name, score outside eps - both sides differ",
			args: args{
				a: []depsmanager.Dependency{
					{Name: "x", Score: 1.0, UpdatedAt: 100},
				},
				b: []depsmanager.Dependency{
					{Name: "x", Score: 1.00001, UpdatedAt: 100},
				},
			},
			wantOnlyA: []depsmanager.Dependency{
				{Name: "x", Score: 1.0, UpdatedAt: 100},
			},
			wantOnlyB: []depsmanager.Dependency{
				{Name: "x", Score: 1.00001, UpdatedAt: 100},
			},
		},
		{
			name: "mix: one equal, one onlyA, one onlyB, one mismatch",
			args: args{
				a: []depsmanager.Dependency{
					{Name: "same", Score: 2.0, UpdatedAt: 20},  // equal
					{Name: "onlyA", Score: 3.0, UpdatedAt: 30}, // only in A
					{Name: "m", Score: 4.0, UpdatedAt: 40},     // mismatched with b
				},
				b: []depsmanager.Dependency{
					{Name: "same", Score: 2.0, UpdatedAt: 20},  // equal
					{Name: "onlyB", Score: 5.0, UpdatedAt: 50}, // only in B
					{Name: "m", Score: 4.1, UpdatedAt: 40},     // mismatched
				},
			},
			wantOnlyA: []depsmanager.Dependency{
				{Name: "onlyA", Score: 3.0, UpdatedAt: 30},
				{Name: "m", Score: 4.0, UpdatedAt: 40},
			},
			wantOnlyB: []depsmanager.Dependency{
				{Name: "onlyB", Score: 5.0, UpdatedAt: 50},
				{Name: "m", Score: 4.1, UpdatedAt: 40},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOnlyA, gotOnlyB := DiffDependencies(tt.args.a, tt.args.b)

			if !reflect.DeepEqual(gotOnlyA, tt.wantOnlyA) {
				t.Fatalf("onlyA mismatch:\n got=%v\nwant=%v", gotOnlyA, tt.wantOnlyA)
			}
			if !reflect.DeepEqual(gotOnlyB, tt.wantOnlyB) {
				t.Fatalf("onlyB mismatch:\n got=%v\nwant=%v", gotOnlyB, tt.wantOnlyB)
			}
		})
	}
}
