package handlers

import (
	"strings"
	"testing"
)

type validationRulesProbe struct {
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required,min=8"`
	Name     string   `json:"name" validate:"max=5"`
	Degree   *int     `json:"degree" validate:"omitempty,min=1"`
	Target   string   `json:"target" validate:"omitempty,uuid"`
	IDs      []string `json:"ids" validate:"max=2,dive,required,uuid"`
}

func validProbe() validationRulesProbe {
	return validationRulesProbe{Email: "user@example.com", Password: "password123"}
}

// The DTOs declare min/max/email/uuid rules; a validator that only understands
// "required" and "oneof" lets a one-character password or a 10 MB name through.
func TestValidateRequest_EnforcesDeclaredRules(t *testing.T) {
	zero := 0
	one := 1
	const contactID = "550e8400-e29b-41d4-a716-446655440000"
	cases := []struct {
		name      string
		mutate    func(*validationRulesProbe)
		wantField string
	}{
		{name: "valid baseline", mutate: func(*validationRulesProbe) {}},
		{name: "password shorter than min", mutate: func(p *validationRulesProbe) { p.Password = "x" }, wantField: "password"},
		{name: "min counts characters not bytes", mutate: func(p *validationRulesProbe) { p.Password = "密码密码密码密码" }},
		{name: "malformed email", mutate: func(p *validationRulesProbe) { p.Email = "not-an-email" }, wantField: "email"},
		{name: "email with surrounding whitespace", mutate: func(p *validationRulesProbe) { p.Email = " user@example.com " }},
		{name: "email with display name", mutate: func(p *validationRulesProbe) { p.Email = "Bob <bob@example.com>" }, wantField: "email"},
		{name: "string longer than max", mutate: func(p *validationRulesProbe) { p.Name = "abcdef" }, wantField: "name"},
		{name: "max counts characters not bytes", mutate: func(p *validationRulesProbe) { p.Name = "五个汉字啊" }},
		{name: "omitempty skips nil pointer", mutate: func(p *validationRulesProbe) { p.Degree = nil }},
		{name: "pointer below min", mutate: func(p *validationRulesProbe) { p.Degree = &zero }, wantField: "degree"},
		{name: "pointer at min", mutate: func(p *validationRulesProbe) { p.Degree = &one }},
		{name: "omitempty skips empty uuid", mutate: func(p *validationRulesProbe) { p.Target = "" }},
		{name: "malformed uuid", mutate: func(p *validationRulesProbe) { p.Target = "1234" }, wantField: "target"},
		{name: "well formed uuid", mutate: func(p *validationRulesProbe) { p.Target = contactID }},
		{name: "slice longer than max", mutate: func(p *validationRulesProbe) { p.IDs = []string{contactID, contactID, contactID} }, wantField: "ids"},
		{name: "dive rejects malformed element", mutate: func(p *validationRulesProbe) { p.IDs = []string{contactID, "nope"} }, wantField: "ids"},
		{name: "dive rejects empty element", mutate: func(p *validationRulesProbe) { p.IDs = []string{""} }, wantField: "ids"},
		{name: "dive accepts valid elements", mutate: func(p *validationRulesProbe) { p.IDs = []string{contactID, contactID} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			probe := validProbe()
			tc.mutate(&probe)

			err := validateRequest(probe)

			if tc.wantField == "" {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected a validation error for %q, got none", tc.wantField)
			}
			if !strings.HasPrefix(err.Error(), tc.wantField+" ") && !strings.HasPrefix(err.Error(), tc.wantField+"[") {
				t.Fatalf("error %q does not name field %q", err.Error(), tc.wantField)
			}
		})
	}
}

type validationRulesNestedProbe struct {
	Pages []validationRulesNestedPage `json:"pages" validate:"required,min=1,dive"`
}

type validationRulesNestedPage struct {
	Name string `json:"name" validate:"required,max=3"`
}

func TestValidateRequest_DiveValidatesNestedStructs(t *testing.T) {
	if err := validateRequest(validationRulesNestedProbe{}); err == nil {
		t.Fatal("an empty pages list passed min=1")
	}
	err := validateRequest(validationRulesNestedProbe{Pages: []validationRulesNestedPage{{Name: "ok"}, {Name: "toolong"}}})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("nested element error = %v, want it to name the nested field", err)
	}
	if err := validateRequest(&validationRulesNestedProbe{Pages: []validationRulesNestedPage{{Name: "ok"}}}); err != nil {
		t.Fatalf("valid nested request rejected: %v", err)
	}
}
