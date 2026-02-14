package handlers

import (
	"fmt"
	"reflect"
	"strings"
)

func validateRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	t := reflect.TypeOf(req)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if rule == "required" && value.Kind() == reflect.String && value.String() == "" {
				jsonTag := field.Tag.Get("json")
				name := strings.Split(jsonTag, ",")[0]
				if name == "" {
					name = field.Name
				}
				return fmt.Errorf("%s is required", name)
			}
		}
	}
	return nil
}
