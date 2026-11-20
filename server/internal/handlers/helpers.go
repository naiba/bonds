package handlers

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// validateRequest 按 DTO 上的 validate 标签校验请求体。
//
// 这里必须实现 DTO 实际用到的每一种规则。早期版本只认识 required 和 oneof，
// 其余规则（min、max、email、uuid、omitempty、dive）写在标签上却从不生效：
// 单字符密码可以注册，名称长度、批量操作的数量上限也都形同虚设。
// 新增一种标签规则时，先在 applyValidationRule 里实现它。
func validateRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" {
			name = field.Name
		}
		if err := validateValue(name, v.Field(i), strings.Split(tag, ",")); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(name string, value reflect.Value, rules []string) error {
	for i, rule := range rules {
		switch rule {
		case "omitempty":
			if value.IsZero() {
				return nil
			}
		case "dive":
			return validateElements(name, value, rules[i+1:])
		default:
			if err := applyValidationRule(name, value, rule); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateElements 实现 dive：后续规则作用于切片的每个元素；
// 元素是结构体且没有后续规则时，按元素自身的 validate 标签递归校验。
func validateElements(name string, value reflect.Value, elementRules []string) error {
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return nil
	}
	for i := 0; i < value.Len(); i++ {
		element := value.Index(i)
		elementName := fmt.Sprintf("%s[%d]", name, i)
		if err := validateValue(elementName, element, elementRules); err != nil {
			return err
		}
		if !element.CanInterface() {
			continue
		}
		if err := validateRequest(element.Interface()); err != nil {
			return fmt.Errorf("%s.%w", elementName, err)
		}
	}
	return nil
}

func applyValidationRule(name string, value reflect.Value, rule string) error {
	switch {
	case rule == "required":
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
	case strings.HasPrefix(rule, "oneof="):
		allowed := strings.Split(strings.TrimPrefix(rule, "oneof="), " ")
		actual := fmt.Sprintf("%v", value.Interface())
		for _, a := range allowed {
			if a == actual {
				return nil
			}
		}
		return fmt.Errorf("%s must be one of [%s]", name, strings.Join(allowed, ", "))
	case strings.HasPrefix(rule, "min="):
		return validateBound(name, value, strings.TrimPrefix(rule, "min="), true)
	case strings.HasPrefix(rule, "max="):
		return validateBound(name, value, strings.TrimPrefix(rule, "max="), false)
	case rule == "email":
		if value.Kind() != reflect.String {
			return nil
		}
		// 首尾空白不算格式错误：服务层会 TrimSpace，手机输入法补全常带一个尾随空格。
		candidate := strings.TrimSpace(value.String())
		address, err := mail.ParseAddress(candidate)
		// ParseAddress 也接受 "Name <a@b>" 形式，登录邮箱必须是裸地址。
		if err != nil || address.Address != candidate {
			return fmt.Errorf("%s must be a valid email address", name)
		}
	case rule == "uuid":
		if value.Kind() != reflect.String {
			return nil
		}
		// uuid.Parse 还接受带花括号、urn: 前缀和无连字符的写法，这里只认标准的 36 位形式。
		if len(value.String()) != 36 || uuid.Validate(value.String()) != nil {
			return fmt.Errorf("%s must be a valid UUID", name)
		}
	}
	return nil
}

// validateBound 实现 min / max：字符串按字符数，切片和 map 按元素个数，数字按数值。
// nil 指针不参与比较（是否必填由 required 决定）。
func validateBound(name string, value reflect.Value, rawLimit string, isMin bool) error {
	limit, err := strconv.ParseFloat(rawLimit, 64)
	if err != nil {
		return nil
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	var actual float64
	var unit string
	switch value.Kind() {
	case reflect.String:
		actual, unit = float64(utf8.RuneCountInString(value.String())), " characters"
	case reflect.Slice, reflect.Array, reflect.Map:
		actual, unit = float64(value.Len()), " items"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		actual = float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		actual = float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		actual = value.Float()
	default:
		return nil
	}

	if isMin && actual < limit {
		return fmt.Errorf("%s must be at least %s%s", name, rawLimit, unit)
	}
	if !isMin && actual > limit {
		return fmt.Errorf("%s must be at most %s%s", name, rawLimit, unit)
	}
	return nil
}
