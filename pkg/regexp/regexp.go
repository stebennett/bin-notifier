package regexputil

import "regexp"

func FindNamedMatches(re *regexp.Regexp, input string) map[string]string {
	match := re.FindStringSubmatch(input)
	result := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return result
}
