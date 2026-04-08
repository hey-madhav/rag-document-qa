package evaluation

import "strings"

// KeywordHitRate returns the fraction of expected keywords found in the answer.
// Simple, deterministic, zero-dependency evaluation metric.
func KeywordHitRate(answer string, expectedKeywords []string) float64 {
	if len(expectedKeywords) == 0 {
		return 0
	}

	lower := strings.ToLower(answer)
	hits := 0
	for _, kw := range expectedKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			hits++
		}
	}
	return float64(hits) / float64(len(expectedKeywords))
}
