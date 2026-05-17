package dto

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// NullableUint is a tri-state uint field for JSON:
//   - Field absent from the JSON body  -> {Present:false, Valid:false}
//   - Field present and null           -> {Present:true,  Valid:false}
//   - Field present and a number       -> {Present:true,  Valid:true, Value:N}
//
// A plain *uint cannot express the distinction because encoding/json
// decodes both `null` and an absent field as a nil pointer, leaving
// callers unable to ask "is this a clear vs a leave-unchanged?".
type NullableUint struct {
	Present bool
	Valid   bool
	Value   uint
}

func (n NullableUint) Set() bool     { return n.Present && n.Valid }
func (n NullableUint) Cleared() bool { return n.Present && !n.Valid }

func (n NullableUint) Ptr() *uint {
	if !n.Set() {
		return nil
	}
	v := n.Value
	return &v
}

func (n *NullableUint) UnmarshalJSON(data []byte) error {
	n.Present = true
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		n.Valid = false
		n.Value = 0
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		if s == "" {
			n.Valid = false
			return nil
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		n.Valid = true
		n.Value = uint(v)
		return nil
	}
	var v uint64
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return err
	}
	n.Valid = true
	n.Value = uint(v)
	return nil
}

func (n NullableUint) MarshalJSON() ([]byte, error) {
	if !n.Set() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatUint(uint64(n.Value), 10)), nil
}
