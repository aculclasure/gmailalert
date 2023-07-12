package cli

import (
	"fmt"

	"github.com/aculclasure/gmailalert/internal/core/processor"
	"golang.org/x/sync/errgroup"
)

type FindEmailsUseCase interface {
	Run(query processor.EmailQuery) (processor.EmailQueryResult, error)
}

type SendAlertUseCase interface {
	Run(alt processor.Alert) error
}

type Logger interface {
	Printf(string, ...interface{})
}

type Processor struct {
	EmailFinder FindEmailsUseCase
	AlertSender SendAlertUseCase
	Logger      Logger
}

func (p *Processor) Process(alerts []Alert) error {
	var errGrp errgroup.Group
	for _, alert := range alerts {
		alert := alert
		errGrp.Go(func() error {
			err := alert.OK()
			if err != nil {
				return err
			}
			queryResult, err := p.EmailFinder.Run(processor.EmailQuery{
				SearchExpression: alert.GmailQuery,
			})
			if err != nil {
				return err
			}
			alert.PushoverMsg = fmt.Sprintf(`Found %d emails matching query "%s"`,
				len(queryResult.MatchingEmails), alert.GmailQuery)
			p.Logger.Printf("%s", alert.PushoverMsg)
			if !processor.AlarmOnResult(queryResult) {
				p.Logger.Printf(`query result "%+v" did not result in an alarm condition`, queryResult)
				return nil
			}
			err = p.AlertSender.Run(processor.Alert{
				Message:   alert.PushoverMsg,
				Title:     alert.PushoverTitle,
				Recipient: alert.PushoverTarget,
			})
			if err != nil {
				return err
			}
			fmt.Printf(`notification titled "%s" successfully sent`, alert.PushoverTitle)
			return nil
		})
	}
	err := errGrp.Wait()
	return err
	// wg := sync.WaitGroup{}
	// wg.Add(len(alerts))
	// for _, alert := range alerts {
	// 	go func(alt Alert) {
	// 		defer wg.Done()
	// 		err := alt.OK()
	// 		if err != nil {
	// 			p.Logger.Printf("got error processing alert %+v: %v", alt, err)
	// 			return
	// 		}
	// 		queryResult, err := p.EmailFinder.Run(processor.EmailQuery{
	// 			SearchExpression: alt.GmailQuery,
	// 		})
	// 		if err != nil {
	// 			p.Logger.Printf("got error searching for email matches: %v", err)
	// 			return
	// 		}
	// 		alt.PushoverMsg = fmt.Sprintf(`Found %d emails matching query "%s"`,
	// 			len(queryResult.MatchingEmails), alt.GmailQuery)
	// 		p.Logger.Printf("%s", alt.PushoverMsg)
	// 		if !processor.AlarmOnResult(queryResult) {
	// 			return
	// 		}
	// 		err = p.AlertSender.Run(processor.Alert{
	// 			Message:   alt.PushoverMsg,
	// 			Title:     alt.PushoverTitle,
	// 			Recipient: alt.PushoverTarget,
	// 		})
	// 		if err != nil {
	// 			p.Logger.Printf("got error sending notification: %v", err)
	// 			return
	// 		}
	// 		fmt.Printf(`notification titled "%s" successfully sent`, alt.PushoverTitle)
	// 	}(alert)
	// }
	// wg.Wait()
	// return nil
}
