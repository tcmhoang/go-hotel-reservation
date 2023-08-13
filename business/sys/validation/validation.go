// Package validation supports for validating nodels
package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var validate *validator.Validate

var translator ut.Translator

func init() {
	validate := validator.New()
	translator, _ = ut.New(en.New(), en.New()).GetTranslator("en")

	en_translations.RegisterDefaultTranslations(validate, translator)

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

func Check(val any) error {
	if err := validate.Struct(val); err != nil {
		verrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return err
		}

		var ferrs FieldErrors

		for _, verr := range verrors {
			ferr := FieldError{
				Field: verr.Field(),
				Error: verr.Error(),
			}

			ferrs = append(ferrs, ferr)
		}
		return ferrs

	}
	return nil
}
