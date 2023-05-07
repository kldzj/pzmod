package util

func Dedupe(list []string) []string {
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(list))
	for _, item := range list {
		if _, ok := seen[item]; !ok {
			seen[item] = true
			deduped = append(deduped, item)
		}
	}

	return deduped
}

func IndexOf(list []string, item string) int {
	for idx, val := range list {
		if val == item {
			return idx
		}
	}

	return -1
}

func Contains(list []string, item string) bool {
	return IndexOf(list, item) >= 0
}

func Filter(list []string, filter func(string) bool) []string {
	filtered := make([]string, 0)
	for _, item := range list {
		if filter(item) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}
