package memtable

import (
	"reflect"
	"sort"
	"testing"
)

func TestPut(t *testing.T) {
	table := NewMemTable()
	val, res := table.Get("sakar")

	if val != "" {
		t.Errorf("Expected empty string, got %s ", val)
	}

	if res == true {
		t.Errorf("Expected false, got %v ", res)
	}
}

func TestPut2(t *testing.T) {
	table := NewMemTable()

	table.Put("sakar", "test")

	res, val := table.Get("sakar")

	if val == false {
		t.Errorf("Expected true , got %v", val)

		if res == "" {
			t.Errorf("Expected a value ,got %s", res)
		}

	}
}

func TestGet(t *testing.T) {
	table := NewMemTable()

	table.Put("sakar", "test")
	table.Put("pahul", "test")
	table.Put("rushat", "test")

	soln := []string{}

	table.Iterate(func(row *MemTableRow) {
		soln = append(soln, row.Key)
	})

	if len(soln) != 3 {
		t.Errorf("Expected 3, got %d", len(soln))
	}

	copy := append([]string{}, soln...)
	sort.Strings(soln)

	if !reflect.DeepEqual(copy, soln) {
		t.Errorf("expected %v, got %v", soln, copy)
	}
}
