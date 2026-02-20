package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

var templateCharacterAliases = map[string]int{
	"blank":             0,
	"space":             0,
	"exclamation":       37,
	"exclamation mark":  37,
	"at":                38,
	"pound":             39,
	"hash":              39,
	"dollar":            40,
	"left parenthesis":  41,
	"open parenthesis":  41,
	"right parenthesis": 42,
	"close parenthesis": 42,
	"hyphen":            44,
	"dash":              44,
	"plus":              46,
	"ampersand":         47,
	"equal":             48,
	"equals":            48,
	"semicolon":         49,
	"colon":             50,
	"single quote":      52,
	"apostrophe":        52,
	"double quote":      53,
	"percent":           54,
	"comma":             55,
	"period":            56,
	"dot":               56,
	"slash":             59,
	"forward slash":     59,
	"question":          60,
	"question mark":     60,
	"degree":            62,
	"heart":             62,
	"red":               63,
	"orange":            64,
	"yellow":            65,
	"green":             66,
	"blue":              67,
	"violet":            68,
	"purple":            68,
	"white":             69,
	"black":             70,
	"filled":            71,
}

func substituteTemplateCharacterAliases(input string) string {
	var out strings.Builder
	out.Grow(len(input))

	for i := 0; i < len(input); {
		if strings.HasPrefix(input[i:], "{{") {
			end := strings.Index(input[i+2:], "}}")
			if end == -1 {
				out.WriteString(input[i:])
				break
			}
			end += i + 4
			out.WriteString(input[i:end])
			i = end
			continue
		}

		if input[i] != '{' {
			out.WriteByte(input[i])
			i++
			continue
		}

		end := strings.IndexByte(input[i+1:], '}')
		if end == -1 {
			out.WriteString(input[i:])
			break
		}
		end += i + 1

		token := strings.TrimSpace(input[i+1 : end])
		out.WriteString(rewriteTemplateToken(token))
		i = end + 1
	}

	return out.String()
}

func rewriteTemplateToken(token string) string {
	if token == "" {
		return "{}"
	}
	if _, err := strconv.Atoi(token); err == nil {
		return fmt.Sprintf("{%s}", token)
	}

	key := canonicalAlias(token)
	if code, ok := templateCharacterAliases[key]; ok {
		return fmt.Sprintf("{%d}", code)
	}
	return fmt.Sprintf("{%s}", token)
}

func canonicalAlias(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	lower = strings.NewReplacer("_", " ", "-", " ").Replace(lower)
	return strings.Join(strings.Fields(lower), " ")
}
