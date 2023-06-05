package report

import (
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var (
	gtwVersionRegex = regexp.MustCompile(`^(v\d+\.\d+\.\d+(\-\d+\-g[a-f0-9]{7})?)$`)
	fpRegex         = regexp.MustCompile(`^[a-f0-9]{32}$`)
	markRegex       = regexp.MustCompile(`^(N/A|[A-F][\+\-]?)$`)
	suffixRegex     = regexp.MustCompile(`^(na|[a-f])$`)
	indicatorRegex  = regexp.MustCompile(`^(-|[[:print:]]{1,30} \((unavailable|[0-9]{1,3}\.[0-9]%)\))$`)
	argsRegex       = regexp.MustCompile(`^\-\-((quiet|tlsVerify|followCookies|renewSession|skipWAFIdentification|nonBlockedAsPassed|noEmailReport|ignoreUnresolved|blockConnReset|skipWAFBlockCheck|addDebugHeader|includePayloads)|(configPath|logFormat|url|wsURL|graphqlURL|proxy|blockRegex|passRegex|testCase|testSet|reportPath|reportName|reportFormat|email|testCasesPath|wafName|addHeader|openapiFile)\=[[:print:]]+|(grpcPort|maxIdleConns|maxRedirects|idleConnTimeout|workers|sendDelay|randomDelay)\=\d+|(blockStatusCodes|passStatusCodes)\=[\d,]+)$`)
)

var customValidators = map[string]validator.Func{
	"gtw_version":  validateGtwVersion,
	"fp":           validateFp,
	"mark":         validateMark,
	"css_suffix":   validateCssSuffix,
	"indicator":    validateIndicator,
	"args":         validateArgs,
	"encoders":     validateEncoders,
	"placeholders": validatePlaceholders,
}

func validateGtwVersion(fl validator.FieldLevel) bool {
	version := fl.Field().String()

	// skip validation if empty or 'unknown' version
	if version == "" || version == "unknown" {
		return true
	}

	result := gtwVersionRegex.MatchString(version)

	return result
}

func validateFp(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	result := fpRegex.MatchString(version)

	return result
}

func validateMark(fl validator.FieldLevel) bool {
	mark := fl.Field().String()
	result := markRegex.MatchString(mark)

	return result
}

func validateCssSuffix(fl validator.FieldLevel) bool {
	suffix := fl.Field().String()
	result := suffixRegex.MatchString(suffix)

	return result
}

func validateIndicator(fl validator.FieldLevel) bool {
	indicator := fl.Field().String()
	result := indicatorRegex.MatchString(indicator)

	return result
}

func validateArgs(fl validator.FieldLevel) bool {
	args := fl.Field().String()
	result := argsRegex.MatchString(args)

	return result
}

func validateEncoders(fl validator.FieldLevel) bool {
	encoders := fl.Field().MapKeys()

	if len(encoders) == 0 {
		return false
	}

	if _, ok := encoders[0].Interface().(string); !ok {
		return false
	}

	for _, e := range encoders {
		if _, ok := encoder.Encoders[e.String()]; !ok {
			return false
		}
	}

	return true
}

func validatePlaceholders(fl validator.FieldLevel) bool {
	placeholders := fl.Field().MapKeys()

	if len(placeholders) == 0 {
		return false
	}

	if _, ok := placeholders[0].Interface().(string); !ok {
		return false
	}

	for _, p := range placeholders {
		if _, ok := placeholder.Placeholders[p.String()]; !ok {
			return false
		}
	}

	return true
}

// ValidateReportData validates report data
func ValidateReportData(reportData *HtmlReport) error {
	validate := validator.New()
	for tag, validatorFunc := range customValidators {
		err := validate.RegisterValidation(tag, validatorFunc)
		if err != nil {
			return errors.Wrap(err, "couldn't build validator")
		}
	}

	err := validate.Struct(reportData)
	if err != nil {
		var validatorErr validator.ValidationErrors
		if errors.As(err, &validatorErr) {
			return &ValidationError{validatorErr}
		}

		return errors.Wrap(err, "couldn't validate report data")
	}

	return nil
}
