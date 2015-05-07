package jsonpatch

import (
	"reflect"
	"testing"
)

type testUser struct {
	Name   string
	Age    int
	Email  string
	Child  *testUser
	Phones []string
	M      map[string]string
}

func TestBestMatch(t *testing.T) {
	type Test struct {
		A string `json:"AwesomeName"`
	}

	ty := reflect.TypeOf(Test{})
	if name := bestMatch("AwesomeName", ty); name != "A" {
		t.Fatal("best match did not work", name)
	}
}

func TestAdd(t *testing.T) {
	u := testUser{
		Phones: []string{"37489239"},
	}
	p := []byte(`[
		{"op": "add", "path": "/name", "value": "Calvin"},
		{"op": "add", "path": "/age", "value": 6},
		{"op": "add", "path": "/email", "value": "calvin@hobbes.com"},
		{"op": "add", "path": "/child/name", "value": "mr. bunny"},
		{"op": "add", "path": "/phones/-", "value": "8390240670"},
                {"op": "add", "path": "/phones/1", "value": "9096040676"},
		{"op": "add", "path": "/m/a", "value": "hello world"}
	]`)
	err := Apply(p, &u)
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "Calvin" {
		t.Fatal("name not set")
	}
	if u.Age != 6 {
		t.Fatal("age not set")
	}
	if u.Email != "calvin@hobbes.com" {
		t.Fatal("email not set")
	}
	if u.Child.Name != "mr. bunny" {
		t.Fatal("pointer value not set")
	}
	if len(u.Phones) != 3 || u.Phones[2] != "8390240670" {
		t.Fatal("slice not set")
	}
	if u.Phones[1] != "9096040676" {
		t.Fatal("slice not set")
	}
	if val, ok := u.M["a"]; !ok || val != "hello world" {
		t.Fatal("map value not set")
	}
}

func TestAdd2(t *testing.T) {
	u := testUser{}
	p := []byte(`[{"op": "add", "path": "/child", "value": {"name": "hobbes"}}]`)
	err := Apply(p, &u)
	if err != nil {
		t.Fatal(err)
	}
	if u.Child == nil {
		t.Fatal("child not initialized")
	}
	if u.Child.Name != "hobbes" {
		t.Fatal("setting child failed", u.Child)
	}
}

func TestAddSlice(t *testing.T) {
	a := []*testUser{
		&testUser{
			Name: "hobbes",
			Age:  100,
		},
	}
	p := []byte(`[
		{"op": "replace", "path": "/0/name", "value": "Calvin"},
		{"op": "replace", "path": "/0/age", "value": 6}
	]`)
	err := Apply(p, &a)
	if err != nil {
		t.Fatal(err)
	}
	if a[0].Name != "Calvin" || a[0].Age != 6 {
		t.Fatal("patch not set", *a[0])
	}
}

func TestReplace(t *testing.T) {
	u := testUser{
		Name:  "hobbes",
		Age:   100,
		Email: "hobbes@calvin.com",
		Child: &testUser{
			Name: "Susie",
		},
		Phones: []string{"12830921"},
		M:      map[string]string{"a": "hello"},
	}
	p := []byte(`[
		{"op": "replace", "path": "/name", "value": "Calvin"},
		{"op": "replace", "path": "/age", "value": 6},
		{"op": "replace", "path": "/email", "value": "calvin@hobbes.com"},
		{"op": "replace", "path": "/child/name", "value": "mr. bunny"},
		{"op": "replace", "path": "/phones/0", "value": "8390240670"},
		{"op": "replace", "path": "/m/a", "value": "hello world"}
	]`)
	err := Apply(p, &u)
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "Calvin" {
		t.Fatal("name not set")
	}
	if u.Age != 6 {
		t.Fatal("age not set")
	}
	if u.Email != "calvin@hobbes.com" {
		t.Fatal("email not set")
	}
	if u.Child.Name != "mr. bunny" {
		t.Fatal("pointer value not set")
	}
	if len(u.Phones) != 1 || u.Phones[0] != "8390240670" {
		t.Fatal("slice not set")
	}
	if val, ok := u.M["a"]; !ok || val != "hello world" {
		t.Fatal("map value not set")
	}
	p = []byte(`{"op": "replace", "path": "/m/a", "value": "hello world"}`)
	err = Apply(p, &u)
	if err == nil {
		t.Fatal("was supposed to fail", u.M)
	}
}

func TestRemove(t *testing.T) {
	u := testUser{
		Name:  "hobbes",
		Age:   100,
		Email: "hobbes@calvin.com",
		Child: &testUser{
			Name: "Susie",
		},
		Phones: []string{"12830921"},
		M:      map[string]string{"a": "hello"},
	}
	p := []byte(`[
		{"op": "remove", "path": "/name"},
		{"op": "remove", "path": "/age"},
		{"op": "remove", "path": "/email"},
		{"op": "remove", "path": "/child/name"},
		{"op": "remove", "path": "/phones/0"},
		{"op": "remove", "path": "/m/a"}
	]`)
	err := Apply(p, &u)
	if err != nil {
		t.Fatal(err)
	}
	if u.Name == "hobbes" {
		t.Fatal("name not removed")
	}
	if u.Age == 100 {
		t.Fatal("age not removed")
	}
	if u.Email == "hobbes@calvin.com" {
		t.Fatal("email not removed")
	}
	if u.Child.Name == "Susie" {
		t.Fatal("pointer value not removed")
	}
	if len(u.Phones) == 1 {
		t.Fatal("slice element not removed")
	}
	if val, ok := u.M["a"]; ok && val != "" {
		t.Fatal("map value not removed", val)
	}
}

func TestTest(t *testing.T) {
	u := testUser{
		Name:  "hobbes",
		Age:   100,
		Email: "hobbes@calvin.com",
		Child: &testUser{
			Name: "Susie",
		},
		Phones: []string{"8390240670"},
		M:      map[string]string{"a": "hello"},
	}
	p := []byte(`[
		{"op": "test", "path": "/name", "value": "hobbes"},
		{"op": "test", "path": "/age", "value": 100},
		{"op": "test", "path": "/email", "value": "hobbes@calvin.com"},
		{"op": "test", "path": "/child/name", "value": "Susie"},
		{"op": "test", "path": "/phones/0", "value": "8390240670"},
		{"op": "test", "path": "/m/a", "value": "hello"}
	]`)
	err := Apply(p, &u)
	if err != nil {
		t.Fatal(err)
	}
}
