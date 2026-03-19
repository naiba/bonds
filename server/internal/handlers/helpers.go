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

		jsonTag := field.Tag.Get("json")
		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			name = field.Name
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if rule == "required" {
				switch value.Kind() {
				case reflect.String:
					if value.String() == "" {
						return fmt.Errorf("%s is required", name)
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if value.Int() == 0 {
						return fmt.Errorf("%s is required", name)
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					if value.Uint() == 0 {
						return fmt.Errorf("%s is required", name)
					}
				}
			}
			if strings.HasPrefix(rule, "oneof=") {
				allowed := strings.Split(strings.TrimPrefix(rule, "oneof="), " ")
				actual := fmt.Sprintf("%v", value.Interface())
				found := false
				for _, a := range allowed {
					if a == actual {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("%s must be one of [%s]", name, strings.Join(allowed, ", "))
				}
			}
		}
	}
	return nil
}
