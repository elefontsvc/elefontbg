package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

var buildVersion string

const svcName = "elefontbg"

var elog debug.Log

type elefontService struct{}

func showUsage(err string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"	where <command> is one of\n"+
			"	install, remove, debug, start, stop\n",
		err, os.Args[0])
	os.Exit(2)
}

func main() {

	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("could not determine if application is running as an interactive session (%v)", err)
	}

	if !isIntSess {
		runSvc(svcName, false)
		return
	}

	if len(os.Args) < 2 {
		showUsage("no command specified")
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		log.Printf("debugging")
		runSvc(svcName, true) //skip the debug run during development, for some reason it wont listen to sigterms
		// h := &http.Server{Addr: "0.0.0.0:42135"}
		// http.HandleFunc("/ws", wsHandler)
		// h.ListenAndServe()
		return
	case "install":
		log.Printf("installing")
		err = installSvc(svcName, "EleFont Background Service")
	case "remove":
		log.Printf("removing")
		err = uninstallSvc(svcName)
	case "start":
		log.Printf("starting")
		err = startSvc(svcName)
	case "stop":
		log.Printf("stopping")
		err = controlSvc(svcName, svc.Stop, svc.Stopped)
	default:
		showUsage(fmt.Sprintf("invalid command '%s'", cmd))
	}
}

func runSvc(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			log.Printf("could not open event log for '%s' (%v)", name, err)
			return
		}
	}
	defer elog.Close()
	elog.Info(1, fmt.Sprintf("starting %s service (%s)", name, buildVersion))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &elefontService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}

func controlSvc(name string, c svc.Cmd, to svc.State) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	st, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control %d: %v", c, err)
	}
	timeout := time.Now().Add(10 * time.Second)
	for st.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state %d: %v", c, err)
		}
		time.Sleep(300 * time.Millisecond)
		st, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

func svcExePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

func installSvc(name, desc string) error {
	exepath, err := svcExePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %v", err)
	}
	return nil
}

func uninstallSvc(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %v", err)
	}
	return nil
}

func (e *elefontService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	tick := time.Tick(500 * time.Millisecond)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	//do stuff here, like ws :)

	if err := loadInstalledFonts(); err != nil {
		elog.Error(1, fmt.Sprintf("could not load installed fonts: %v", err))
		changes <- svc.Status{State: svc.StopPending}
		return
	}

	h := &http.Server{Addr: "0.0.0.0:42135"}
	http.HandleFunc("/ws", wsHandler)
	go h.ListenAndServe()

SVCLOOP:
	for {
		select {
		case <-tick:
		//do nothing
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, fmt.Sprintf("received shutdown command, quitting gracefully"))
				h.Shutdown(context.Background())
				elog.Info(1, fmt.Sprintf("elefont background service shutdown gracefully"))
				break SVCLOOP
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request %d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func startSvc(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	err = s.Start("is", "manual-started")
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	return nil
}
