package quiz

import "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"

// scoreQuestion returns points earned and points possible for one question.
//
// TRUE_FALSE: 1 point when the single selected option equals the key, else 0.
// MULTIPLE_CHOICE: one point for every option position where the student's mark
// equals the key. Marked-and-correct and unmarked-and-not-correct both count.
// Key 0110 with answer 0101 matches positions 1 and 2, so 2 points of 4.
//
// A question with nothing selected (or never answered) always earns 0 points, but its
// points still count as possible. Key 0010 with nothing selected (0000) is 0 of 4, not 3 of 4.
func scoreQuestion(qType content.QuestionType, optionCount int, key, selected []int) (earned, possible int) {
	if qType == content.TypeTrueFalse {
		if sameSet(key, selected) {
			return 1, 1
		}
		return 0, 1
	}
	if len(selected) == 0 {
		return 0, optionCount
	}

	keySet := toSet(key)
	selectedSet := toSet(selected)
	for i := 0; i < optionCount; i++ {
		if keySet[i] == selectedSet[i] {
			earned++
		}
	}
	return earned, optionCount
}

// percentOf returns earned/possible as a percentage. Zero possible gives 0.
func percentOf(earned, possible int) float64 {
	if possible == 0 {
		return 0
	}
	return float64(earned) * 100 / float64(possible)
}

func toSet(values []int) map[int]bool {
	set := make(map[int]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

func sameSet(a, b []int) bool {
	as, bs := toSet(a), toSet(b)
	if len(as) != len(bs) {
		return false
	}
	for k := range as {
		if !bs[k] {
			return false
		}
	}
	return true
}
