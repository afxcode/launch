package main

import (
	"fmt"
	"time"

	l "launch/internal/pkg/logger"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

func winservice(signalQuit, signalQuited, signalDone chan bool, appName string) bool {
	if isService, _ := svc.IsWindowsService(); !isService {
		return false
	}

	go func() {
		elog, err := eventlog.Open(appName)
		if err != nil {
			return
		}
		defer elog.Close()

		elog.Info(1, fmt.Sprintf("Starting %s service", appName))
		s := &wservice{signalQuit, signalQuited}
		if err = svc.Run(appName, s); err != nil {
			l.W(err.Error())
		}
		elog.Info(1, fmt.Sprintf("%s service stopped", appName))
		l.I("Windows Service: send signal done")
		signalDone <- true
		l.I("Windows Service: sent signal done")
	}()
	return true
}

type wservice struct {
	signalQuit   chan bool
	signalQuited chan bool
}

func (s *wservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	time.Sleep(100 * time.Millisecond)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				l.E(fmt.Sprintf("Windows Service: unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	l.I("Windows Service: send signal quit")
	s.signalQuit <- true // send signal to grace stop
	l.I("Windows Service: sent signal quit, wait quited")

	select {
	case <-s.signalQuited: // wait for grace stop
		changes <- svc.Status{State: svc.Stopped}
		l.I("Windows Service: received signal quited")
	case <-time.After(30 * time.Second):
		l.I("Windows Service: wait quited timeout, quited itself")
	}
	return
}
