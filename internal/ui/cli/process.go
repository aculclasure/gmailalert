package cli

import (
	"fmt"
	"sync/atomic"

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
	var (
		errGrp           errgroup.Group
		numEmittedAlerts uint64
	)
	fmt.Printf("Processing %d email queries to determine if any alerts will be emitted...\n", len(alerts))
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
			alert.PushoverMsg = fmt.Sprintf(`found %d emails matching query "%s"`,
				len(queryResult.MatchingEmails), alert.GmailQuery)
			if !processor.AlarmOnResult(queryResult) {
				p.Logger.Printf(`query result "%+v" did not result in an alarm condition`, queryResult)
				return nil
			}
			alt := processor.Alert{
				Message:   alert.PushoverMsg,
				Title:     alert.PushoverTitle,
				Recipient: alert.PushoverTarget,
				Sound:     alert.PushoverSound,
			}
			p.Logger.Printf("sending alert %+v\n", alt)
			err = p.AlertSender.Run(alt)
			if err != nil {
				return err
			}
			atomic.AddUint64(&numEmittedAlerts, 1)
			p.Logger.Printf("successfully sent alert %+v\n", alt)
			fmt.Printf("Alert titled \"%s\" successfully sent\n", alert.PushoverTitle)
			return nil
		})
	}
	err := errGrp.Wait()
	fmt.Printf("Emitted %d alerts\n", numEmittedAlerts)
	return err
}
