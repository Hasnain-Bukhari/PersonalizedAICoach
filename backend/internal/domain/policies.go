package domain

import (
	"math"
	"time"
)

// ScheduleSM2 applies the product's deliberate SM-2 variant. Low quality is
// reviewed tomorrow, medium quality in seven days, and perfect recall in at
// least fourteen days before normal ease-factor multiplication takes over.
func ScheduleSM2(node KnowledgeNode, quality int, now time.Time) KnowledgeNode {
	if quality < 0 {
		quality = 0
	}
	if quality > 5 {
		quality = 5
	}
	delta := float64(5 - quality)
	node.EaseFactor += 0.1 - delta*(0.08+delta*0.02)
	if node.EaseFactor < 1.3 {
		node.EaseFactor = 1.3
	}
	switch {
	case quality < 3:
		node.Repetitions = 0
		node.LastIntervalDays = 1
	case quality < 5:
		node.Repetitions++
		node.LastIntervalDays = 7
	default:
		node.Repetitions++
		if node.LastIntervalDays < 14 {
			node.LastIntervalDays = 14
		} else {
			node.LastIntervalDays = int(math.Ceil(float64(node.LastIntervalDays) * node.EaseFactor))
		}
	}
	node.Attempts++
	node.LastStudied = now.UTC()
	node.NextRevisionDue = now.UTC().AddDate(0, 0, node.LastIntervalDays)
	return node
}

func UpdateMastery(current, observed float64, attempts int) float64 {
	if observed < 0 {
		observed = 0
	}
	if observed > 100 {
		observed = 100
	}
	weight := .35
	if attempts == 0 {
		weight = .65
	}
	value := current*(1-weight) + observed*weight
	return math.Round(math.Max(0, math.Min(100, value))*100) / 100
}

func QualityFromAnswer(correct bool, confidence float64, skipped bool) int {
	if skipped {
		return 0
	}
	if !correct {
		if confidence >= .7 {
			return 1
		}
		return 2
	}
	if confidence >= .8 {
		return 5
	}
	if confidence >= .4 {
		return 4
	}
	return 3
}
