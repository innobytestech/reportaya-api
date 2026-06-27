package validate

import (
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	v        *validator.Validate
	initOnce sync.Once
)

// V returns a singleton *validator.Validate with all custom validators
// registered. Safe for concurrent use.
func V() *validator.Validate {
	initOnce.Do(initValidator)
	return v
}

func initValidator() {
	v = validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	_ = v.RegisterValidation("sku", validateSKU)
	_ = v.RegisterValidation("rfc_mx", validateRFCMx)
	_ = v.RegisterValidation("phone_mx", validatePhoneMx)
	_ = v.RegisterValidation("strong_password", validateStrongPassword)
	_ = v.RegisterValidation("positive", validatePositive)
	_ = v.RegisterValidation("non_negative", validateNonNegative)
}

var (
	skuRegex     = regexp.MustCompile(`^[A-Z0-9][A-Z0-9\-]{1,48}[A-Z0-9]$`)
	rfcMxRegex   = regexp.MustCompile(`^[A-ZÑ&]{3,4}\d{6}[A-Z\d]{3}$`)
	phoneMxRegex = regexp.MustCompile(`^(\+?52)?\d{10}$`)
)

func validateSKU(fl validator.FieldLevel) bool {
	s := strings.ToUpper(strings.TrimSpace(fl.Field().String()))
	return skuRegex.MatchString(s)
}

func validateRFCMx(fl validator.FieldLevel) bool {
	s := strings.ToUpper(strings.TrimSpace(fl.Field().String()))
	return rfcMxRegex.MatchString(s)
}

func validatePhoneMx(fl validator.FieldLevel) bool {
	s := strings.ReplaceAll(strings.TrimSpace(fl.Field().String()), " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return phoneMxRegex.MatchString(s)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	return Password(fl.Field().String())
}

func validatePositive(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Float32, reflect.Float64:
		return fl.Field().Float() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fl.Field().Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fl.Field().Uint() > 0
	}
	return false
}

func validateNonNegative(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Float32, reflect.Float64:
		return fl.Field().Float() >= 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fl.Field().Int() >= 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

// TranslateFieldError returns a human-readable Spanish message for a single
// validator.FieldError. This is used for developer-facing internal logs; the
// client receives only a generic public message plus an error code.
func TranslateFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " es obligatorio"
	case "email":
		return field + " debe ser un correo válido"
	case "min":
		return field + " no alcanza el mínimo (" + fe.Param() + ")"
	case "max":
		return field + " excede el máximo permitido (" + fe.Param() + ")"
	case "len":
		return field + " debe tener longitud exacta de " + fe.Param()
	case "uuid", "uuid4":
		return field + " debe ser un UUID válido"
	case "oneof":
		return field + " debe ser uno de: " + fe.Param()
	case "sku":
		return field + " no tiene formato válido de SKU"
	case "rfc_mx":
		return field + " no es un RFC válido"
	case "phone_mx":
		return field + " no es un teléfono válido"
	case "strong_password":
		return field + " no cumple la política de contraseñas"
	case "positive":
		return field + " debe ser mayor que cero"
	case "non_negative":
		return field + " no puede ser negativo"
	case "gte":
		return field + " debe ser mayor o igual a " + fe.Param()
	case "lte":
		return field + " debe ser menor o igual a " + fe.Param()
	case "gt":
		return field + " debe ser mayor que " + fe.Param()
	case "lt":
		return field + " debe ser menor que " + fe.Param()
	}
	return field + " no es válido (regla " + fe.Tag() + ")"
}
