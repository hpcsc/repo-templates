package request

import "regexp"

type RouteMatcher func(string) bool

func exactMatcher(s1 string) RouteMatcher {
	return func(s2 string) bool {
		return s1 == s2
	}
}

func patternMatcher(pattern string) RouteMatcher {
	re := regexp.MustCompile(pattern)
	return func(s2 string) bool {
		return re.MatchString(s2)
	}
}

func NewRouteMatcher(ignoreRoutePatterns []string) RouteMatcher {
	routeWithVariables := regexp.MustCompile(`{.*?}`)
	var matchers = make([]RouteMatcher, len(ignoreRoutePatterns))
	for i, routePattern := range ignoreRoutePatterns {
		if routeWithVariables.MatchString(routePattern) {
			anyValuePattern := routeWithVariables.ReplaceAllString(routePattern, "[^/]+?")
			matchers[i] = patternMatcher(anyValuePattern)
		} else {
			matchers[i] = exactMatcher(routePattern)
		}
	}

	return func(path string) bool {
		for _, match := range matchers {
			if match(path) {
				return true
			}
		}

		return false
	}
}
