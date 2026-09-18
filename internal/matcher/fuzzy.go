package matcher

import (
	"math"
	"regexp"
	"strings"
)

var wordRegex = regexp.MustCompile(`[a-zA-Z0-9_\-\.]+`)

// FuzzyScore calculates a similarity score between input and target string in range [0.0, 1.0].
func FuzzyScore(input, target string) float64 {
	input = strings.TrimSpace(strings.ToLower(input))
	target = strings.TrimSpace(strings.ToLower(target))

	if input == target {
		return 1.0
	}
	if len(input) == 0 || len(target) == 0 {
		return 0.0
	}

	// Substring bonus
	if strings.Contains(input, target) || strings.Contains(target, input) {
		shorter := math.Min(float64(len(input)), float64(len(target)))
		longer := math.Max(float64(len(input)), float64(len(target)))
		return 0.70 + 0.25*(shorter/longer)
	}

	tokenSim := tokenJaccard(input, target)
	trigramSim := trigramSimilarity(input, target)
	levSim := levenshteinSimilarity(input, target)

	// Balanced blend: high Levenshtein & Trigram captures typos; token Jaccard captures word reordering
	score := 0.40*levSim + 0.40*trigramSim + 0.20*tokenSim
	return math.Min(1.0, math.Max(0.0, score))
}

func tokenJaccard(s1, s2 string) float64 {
	words1 := wordRegex.FindAllString(s1, -1)
	words2 := wordRegex.FindAllString(s2, -1)
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	intersection := 0
	set2 := make(map[string]bool)
	for _, w := range words2 {
		set2[w] = true
		if set1[w] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func trigramSimilarity(s1, s2 string) float64 {
	tri1 := makeTrigrams(s1)
	tri2 := makeTrigrams(s2)
	if len(tri1) == 0 || len(tri2) == 0 {
		return 0.0
	}

	count1 := make(map[string]int)
	for _, t := range tri1 {
		count1[t]++
	}

	matches := 0
	for _, t := range tri2 {
		if count1[t] > 0 {
			matches++
			count1[t]--
		}
	}

	return float64(2*matches) / float64(len(tri1)+len(tri2))
}

func makeTrigrams(s string) []string {
	runes := []rune(s)
	if len(runes) < 3 {
		return []string{s}
	}
	var trigrams []string
	for i := 0; i <= len(runes)-3; i++ {
		trigrams = append(trigrams, string(runes[i:i+3]))
	}
	return trigrams
}

func levenshteinSimilarity(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return 0.0
	}
	if m == 0 {
		return 0.0
	}

	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
		d[i][0] = i
	}
	for j := 0; j <= m; j++ {
		d[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	dist := d[n][m]
	maxLen := math.Max(float64(n), float64(m))
	return 1.0 - (float64(dist) / maxLen)
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
