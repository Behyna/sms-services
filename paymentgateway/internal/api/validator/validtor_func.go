package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

const (
	priceRegex = `^\d+(,\d{1,2})?$`
)

const (
	AmountTag = "amount"
)

var valid = map[string]func(fl validator.FieldLevel) bool{
	AmountTag: ValidatePrice,
}

func ValidatePrice(fl validator.FieldLevel) bool {
	price := fl.Field().String()
	return regexp.MustCompile(priceRegex).MatchString(price)
}
