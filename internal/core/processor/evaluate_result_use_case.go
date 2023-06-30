package processor

func AlarmOnResult(e EmailQueryResult) bool {
	return len(e.MatchingEmails) > 0
}
