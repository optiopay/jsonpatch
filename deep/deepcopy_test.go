package deep

import (
	"testing"
)

func TestSlice(t *testing.T) {
	a := []string{"hello", "world"}
	var b []string
	err := Copy(&a, &b)
	if err != nil {
		t.Fatal(err)
	}
	for i, x := range b {
		if a[i] != x {
			t.Fatal(i, x)
		}
	}
}

func TestSlicePtr(t *testing.T) {
	type s struct {
		A string
	}
	a := []*s{&s{A: "hello"}}
	var b []*s
	err := Copy(&a, &b)
	if err != nil {
		t.Fatal(err)
	}
	a[0].A = "world"
	if a[0].A == b[0].A {
		t.Fatal(*b[0])
	}
}

func TestMap(t *testing.T) {
	a := map[string]int{"a": 1, "b": 2}
	var b map[string]int
	err := Copy(&a, &b)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range a {
		if x, ok := b[k]; !ok || x != v {
			t.Fatal("copy failed for", k, x)
		}
	}
}

func TestMapPtr(t *testing.T) {
	x := "hello"
	a := map[string]*string{"a": &x}
	var b map[string]*string
	err := Copy(&a, &b)
	if err != nil {
		t.Fatal(err)
	}
	x = "world"
	if *(a["a"]) == *(b["a"]) {
		t.Fatal(*(a["a"]), "should not be equal to", *(b["a"]))
	}
}

func TestStruct(t *testing.T) {
	type s struct {
		A string
		B *s
	}
	a := s{
		A: "hello",
		B: &s{
			A: "world",
		},
	}
	b := s{}
	err := Copy(&a, &b)
	if err != nil {
		t.Fatal(err)
	}
	a.B.A = "hello"
	a.A = "world"
	if a.A == b.A {
		t.Fatal(b)
	}
	if a.B.A == b.B.A {
		t.Fatal(*b.B)
	}
}

func TestPointerToPointer(t *testing.T) {
	type s struct {
		A string
	}

	a := &s{A: "hello"}
	b := &s{}
	err := Copy(&a, &b)
	if err != nil {
		t.Fatal(err)
	}
	if b.A != a.A {
		t.Fatal(*b)
	}
}

// I wonder if anybody would want to do this, still better to test
func TestPrimitives(t *testing.T) {
	sa := "hello"
	sb := ""
	err := Copy(&sa, &sb)
	if err != nil {
		t.Fatal(err)
	}
	if sa != sb {
		t.Fatal(sb, "not the same as", sa)
	}

	ia := 10
	ib := 0
	err = Copy(&ia, &ib)
	if err != nil {
		t.Fatal(err)
	}
	if sa != sb {
		t.Fatal(ib, "not the same as", ia)
	}

	fa := 10.1
	fb := 0.0
	err = Copy(&fa, &fb)
	if err != nil {
		t.Fatal(err)
	}
	if sa != sb {
		t.Fatal(fb, "not the same as", fb)
	}
}
