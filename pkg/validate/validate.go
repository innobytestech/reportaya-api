package validate

import (
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,50}$`)

type PasswordPolicy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSpecial bool
	DisallowSpaces bool
}

func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      12,
		MaxLength:      128,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		DisallowSpaces: true,
	}
}

func PasswordPolicyFromEnv() PasswordPolicy {
	p := DefaultPasswordPolicy()
	p.MinLength = getEnvInt("PASSWORD_MIN_LENGTH", p.MinLength)
	p.MaxLength = getEnvInt("PASSWORD_MAX_LENGTH", p.MaxLength)
	p.RequireUpper = getEnvBool("PASSWORD_REQUIRE_UPPER", p.RequireUpper)
	p.RequireLower = getEnvBool("PASSWORD_REQUIRE_LOWER", p.RequireLower)
	p.RequireDigit = getEnvBool("PASSWORD_REQUIRE_DIGIT", p.RequireDigit)
	p.RequireSpecial = getEnvBool("PASSWORD_REQUIRE_SPECIAL", p.RequireSpecial)
	p.DisallowSpaces = getEnvBool("PASSWORD_DISALLOW_SPACES", p.DisallowSpaces)

	if p.MinLength < 1 {
		p.MinLength = 1
	}
	if p.MaxLength < p.MinLength {
		p.MaxLength = p.MinLength
	}
	return p
}

func PasswordRequirementsText(policy PasswordPolicy) string {
	parts := make([]string, 0, 6)
	parts = append(parts, "password must be "+strconv.Itoa(policy.MinLength)+"-"+strconv.Itoa(policy.MaxLength)+" chars")
	if policy.RequireUpper {
		parts = append(parts, "uppercase")
	}
	if policy.RequireLower {
		parts = append(parts, "lowercase")
	}
	if policy.RequireDigit {
		parts = append(parts, "number")
	}
	if policy.RequireSpecial {
		parts = append(parts, "special character")
	}
	msg := strings.Join(parts[:1], "")
	if len(parts) > 1 {
		msg += " and include " + strings.Join(parts[1:], ", ")
	}
	if policy.DisallowSpaces {
		msg += " (no spaces)"
	}
	return msg
}

func Email(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(v)
	if err != nil {
		return false
	}
	return strings.EqualFold(addr.Address, v)
}

func Username(v string) bool {
	v = strings.TrimSpace(v)
	return usernameRegex.MatchString(v)
}

func Password(v string) bool {
	return PasswordWithPolicy(v, PasswordPolicyFromEnv())
}

func PasswordWithPolicy(v string, policy PasswordPolicy) bool {
	if len(v) < policy.MinLength || len(v) > policy.MaxLength {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range v {
		switch {
		case unicode.IsSpace(r):
			if policy.DisallowSpaces {
				return false
			}
			continue
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	if policy.RequireUpper && !hasUpper {
		return false
	}
	if policy.RequireLower && !hasLower {
		return false
	}
	if policy.RequireDigit && !hasDigit {
		return false
	}
	if policy.RequireSpecial && !hasSpecial {
		return false
	}
	return true
}

func getEnvInt(key string, defaultVal int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func getEnvBool(key string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return defaultVal
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultVal
	}
}
