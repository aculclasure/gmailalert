package processor_test

import (
	"testing"

	"github.com/aculclasure/gmailalert/internal/core/processor"
	"github.com/google/go-cmp/cmp"
)

type mockEmailRepo struct {
	emails []string
}

func (m *mockEmailRepo) Find(searchExpression string) ([]string, error) {
	return m.emails, nil
}

func TestFindEmailsUseCase_RunWithReturnedEmailResultsReturnsExpectedEmailQueryResult(t *testing.T) {
	t.Parallel()
	emailRepo := &mockEmailRepo{
		emails: []string{"email1", "email2", "email3"},
	}
	emailFinder, err := processor.NewFindEmailsUseCase(emailRepo)
	if err != nil {
		t.Fatal(err)
	}
	query := processor.EmailQuery{
		SearchExpression: "is:unread subject:Payment Due!",
		Alert: processor.Alert{
			Sound:       "cashregister",
			Destination: "pagerdutyappid",
		},
	}
	want := processor.EmailQueryResult{
		Query:          query,
		MatchingEmails: []string{"email1", "email2", "email3"},
	}
	got, err := emailFinder.Run(query)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}
