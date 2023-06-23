package processor

type Alert struct {
	Sound       string
	Destination string
}

type EmailQuery struct {
	SearchExpression string
	Alert            Alert
}

type EmailQueryResult struct {
	Query          EmailQuery
	MatchingEmails []string
}
