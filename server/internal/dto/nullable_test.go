package dto

import (
	"encoding/json"
	"testing"
)

func TestNullableUintUnmarshalAbsent(t *testing.T) {
	var body struct {
		Other  string       `json:"other"`
		Parent NullableUint `json:"parent_task_id"`
	}
	if err := json.Unmarshal([]byte(`{"other":"x"}`), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Parent.Present {
		t.Errorf("absent field should not be Present, got %+v", body.Parent)
	}
	if body.Parent.Set() || body.Parent.Cleared() {
		t.Errorf("absent should be neither Set nor Cleared")
	}
	if body.Parent.Ptr() != nil {
		t.Errorf("Ptr() should be nil for absent, got %v", body.Parent.Ptr())
	}
}

func TestNullableUintUnmarshalNull(t *testing.T) {
	var body struct {
		Parent NullableUint `json:"parent_task_id"`
	}
	if err := json.Unmarshal([]byte(`{"parent_task_id":null}`), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Parent.Present {
		t.Errorf("null should be Present")
	}
	if body.Parent.Set() {
		t.Errorf("null should not be Set")
	}
	if !body.Parent.Cleared() {
		t.Errorf("null should be Cleared")
	}
	if body.Parent.Ptr() != nil {
		t.Errorf("Ptr() should be nil for cleared")
	}
}

func TestNullableUintUnmarshalNumber(t *testing.T) {
	var body struct {
		Parent NullableUint `json:"parent_task_id"`
	}
	if err := json.Unmarshal([]byte(`{"parent_task_id":42}`), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Parent.Set() {
		t.Errorf("number should be Set, got %+v", body.Parent)
	}
	if body.Parent.Value != 42 {
		t.Errorf("Value = %d, want 42", body.Parent.Value)
	}
	p := body.Parent.Ptr()
	if p == nil || *p != 42 {
		t.Errorf("Ptr() = %v, want *42", p)
	}
}

func TestNullableUintUnmarshalStringNumber(t *testing.T) {
	var body struct {
		Parent NullableUint `json:"parent_task_id"`
	}
	if err := json.Unmarshal([]byte(`{"parent_task_id":"7"}`), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Parent.Set() || body.Parent.Value != 7 {
		t.Errorf("string-encoded number should parse, got %+v", body.Parent)
	}
}

func TestNullableUintUnmarshalEmptyString(t *testing.T) {
	var body struct {
		Parent NullableUint `json:"parent_task_id"`
	}
	if err := json.Unmarshal([]byte(`{"parent_task_id":""}`), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Parent.Present || body.Parent.Valid {
		t.Errorf("empty string should be Present but not Valid, got %+v", body.Parent)
	}
}

func TestNullableUintMarshal(t *testing.T) {
	cases := []struct {
		name string
		in   NullableUint
		want string
	}{
		{"absent", NullableUint{}, `null`},
		{"cleared", NullableUint{Present: true}, `null`},
		{"set", NullableUint{Present: true, Valid: true, Value: 99}, `99`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := json.Marshal(c.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(out) != c.want {
				t.Errorf("got %s, want %s", out, c.want)
			}
		})
	}
}
