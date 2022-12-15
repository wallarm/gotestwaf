package report

import (
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var (
	gtwVersionRegex = regexp.MustCompile(`^(v\d+\.\d+\.\d+(\-\d+\-g[a-f0-9]{7})?|unknown)$`)
	fpRegex         = regexp.MustCompile(`^[a-f0-9]{32}$`)
	markRegex       = regexp.MustCompile(`^(N/A|[A-F][\+\-]?)$`)
	suffixRegex     = regexp.MustCompile(`^(na|[a-f])$`)
	indicatorRegex  = regexp.MustCompile(`^(-|[[:print:]]{1,30} \((unavailable|[0-9]{1,3}\.[0-9]%)\))$`)
	argsRegex       = regexp.MustCompile(`^\-\-((quiet|tlsVerify|followCookies|renewSession|skipWAFIdentification|nonBlockedAsPassed|noEmailReport|ignoreUnresolved|blockConnReset|skipWAFBlockCheck|addDebugHeader|includePayloads)|(configPath|logFormat|url|wsURL|graphqlURL|proxy|blockRegex|passRegex|testCase|testSet|reportPath|reportName|reportFormat|email|testCasesPath|wafName|addHeader|openapiFile)\=[[:print:]]+|(grpcPort|maxIdleConns|maxRedirects|idleConnTimeout|workers|sendDelay|randomDelay)\=\d+|(blockStatusCodes|passStatusCodes)\=[\d,]+)$`)
)

func validateGtwVersion(fl validator.FieldLevel) bool {
	version := fl.Field().String()
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
	validate.RegisterValidation("gtw_version", validateGtwVersion)
	validate.RegisterValidation("fp", validateFp)
	validate.RegisterValidation("mark", validateMark)
	validate.RegisterValidation("css_suffix", validateCssSuffix)
	validate.RegisterValidation("indicator", validateIndicator)
	validate.RegisterValidation("args", validateArgs)
	validate.RegisterValidation("encoders", validateEncoders)
	validate.RegisterValidation("placeholders", validatePlaceholders)

	err := validate.Struct(reportData)
	if err != nil {
		return errors.Wrap(err, "found invalid values in the report data")
	}

	return nil
}
