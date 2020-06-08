package lang

import "fmt"

func candidateSortTextPolicy(candidate CompletionCandidate) string {
	switch candidate.(type) {
	case *AllRequiredFieldCandidate:
		return "00001"
	case *attributeCandidate:
		return "20000"
	case *nestedBlockCandidate:
		return "40000"
	case *labelCandidate:
		return "60000"
	case *completableBlockType:
		return "80000"
	default:
		return "99999"
	}
}

func setCandidateSortText(candidate CompletionCandidate, order int) {
	switch v := candidate.(type) {
	case *AllRequiredFieldCandidate:
		v.sortText = fmt.Sprintf("%05d", order)
	case *attributeCandidate:
		v.sortText = fmt.Sprintf("%05d", order)
	case *nestedBlockCandidate:
		v.sortText = fmt.Sprintf("%05d", order)
	case *labelCandidate:
		v.sortText = fmt.Sprintf("%05d", order)
	case *completableBlockType:
		v.sortText = fmt.Sprintf("%05d", order)
	}
}
