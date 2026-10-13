package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fakeQuerier map[string][]map[string]string

func (f fakeQuerier) QueryRows(sql string) ([]map[string]string, error) {
	rows, ok := f[sql]
	if !ok {
		return nil, errors.New("no such table")
	}
	return rows, nil
}

func TestReadQueriesSkipsBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queries")
	if err := os.WriteFile(path, []byte("select 1;\n\n  \nselect 2;  \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readQueries(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"select 1;", "select 2;"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("readQueries = %q, want %q", got, want)
	}
}

func TestRunQueriesGroupsColumnsAcrossRowsAndQueries(t *testing.T) {
	q := fakeQuerier{
		"a": {{"name": "x", "version": "1"}, {"name": "y"}},
		"b": {{"version": "2"}},
	}

	got, err := runQueries(q, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"name": {"x", "y"}, "version": {"1", "2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runQueries = %v, want %v", got, want)
	}
}

// A failing query must surface as an error rather than hang or yield partial
// results, so the caller never writes a half-updated plist.
func TestRunQueriesFailsOnQueryError(t *testing.T) {
	q := fakeQuerier{"a": {{"name": "x"}}}
	if _, err := runQueries(q, []string{"a", "missing"}); err == nil {
		t.Fatal("runQueries succeeded, want error")
	}
}

// Other condition scripts share ConditionalItems.plist; their keys must
// survive a run, while stale osquery_ keys are replaced.
func TestMergePreservesForeignKeysAcrossSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ConditionalItems.plist")

	conds, err := loadConditions(path)
	if err != nil {
		t.Fatalf("missing plist: %v", err)
	}
	conds["other_script"] = "keep"
	conds["osquery_version"] = []string{"old"}
	if err = saveConditions(path, conds); err != nil {
		t.Fatal(err)
	}

	conds, err = loadConditions(path)
	if err != nil {
		t.Fatal(err)
	}
	conds.merge(map[string][]string{"version": {"new"}})
	if err = saveConditions(path, conds); err != nil {
		t.Fatal(err)
	}

	got, err := loadConditions(path)
	if err != nil {
		t.Fatal(err)
	}
	want := conditions{"other_script": "keep", "osquery_version": []any{"new"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("conditions = %v, want %v", got, want)
	}
}
