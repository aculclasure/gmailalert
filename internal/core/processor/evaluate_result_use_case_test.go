package processor_test

import (
	"testing"

	"github.com/aculclasure/gmailalert/internal/core/processor"
)

func TestAlarmOnResult(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		input processor.EmailQueryResult
		want  bool
	}{
		"Email query result with non-empty matching emails returns true": {
			input: processor.EmailQueryResult{
				MatchingEmails: []string{
					"email matching a search expression",
				},
			},
			want: true,
		},
		"Email query result with no matching emails returns false": {
			input: processor.EmailQueryResult{
				MatchingEmails: []string{},
			},
			want: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := processor.AlarmOnResult(tc.input)
			if tc.want != got {
				t.Errorf("want %t, got %t", tc.want, got)
			}
		})
	}

}
