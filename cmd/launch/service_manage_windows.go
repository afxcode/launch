package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"

	"launch/internal/pkg/errors"
	l "launch/internal/pkg/logger"
)

func serviceInstall(appName, executableFile, executableDir string, enableLogFile bool) (err error) {
	if !amAdmin() {
		return runMeElevated()
	}

	defer func() {
		l.LogError("Service Installation", errors.E(err))
		fmt.Println("Hit Enter to close")
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
	}()

	l.I("Installing service")
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("read executable failed: %w", err)
	}
	fi, err := os.Stat(executable)
	if err == nil {
		if fi.Mode().IsDir() {
			return fmt.Errorf("executable is a directory")
		}
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service connect failed: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(appName)
	if err == nil {
		_ = s.Close()
		return fmt.Errorf("service %s already exist", appName)
	}

	args := []string{
		fmt.Sprintf("-appname=%s", appName),
		fmt.Sprintf("-executable=%s", executableFile),
		fmt.Sprintf("-workdir=%s", executableDir),
	}
	if enableLogFile {
		args = append(args, "-enable-log-file")
	}

	s, err = m.CreateService(
		appName,
		executable,
		mgr.Config{
			StartType:        mgr.StartAutomatic,
			DelayedAutoStart: true,
			DisplayName:      "Launch - " + appName,
			Description:      "Running " + appName + " via Launch",
		},
		args...,
	)

	if err != nil {
		return fmt.Errorf("install service failed: %w", err)
	}
	defer s.Close()

	if err = eventlog.InstallAsEventCreate(appName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		_ = s.Delete()
		return fmt.Errorf("install event log failed: %sw", err)
	}

	return nil
}

func serviceUninstall(appName string) (err error) {
	if !amAdmin() {
		return runMeElevated()
	}

	defer func() {
		l.LogError("Service Uninstallation", errors.E(err))
		fmt.Println("Hit Enter to close")
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
	}()

	l.I("Uninstalling service")
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service connect failed: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(appName)
	if err != nil {
		return fmt.Errorf("open service failed: %w", err)
	}
	defer s.Close()

	if err = stop(s); err != nil {
		return
	}

	if err = s.Delete(); err != nil {
		return fmt.Errorf("remove service failed %s: %w", s.Name, err)
	}

	if err = eventlog.Remove(s.Name); err != nil {
		return fmt.Errorf("remove event log failed: %w", err)
	}
	return nil
}

func serviceStart(appName string) (err error) {
	if !amAdmin() {
		return runMeElevated()
	}

	defer func() {
		l.LogError("Service Start", errors.E(err))
		fmt.Println("Hit Enter to close")
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
	}()

	l.I("Starting service")
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service connect failed: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(appName)
	if err != nil {
		return fmt.Errorf("open service failed: %w", err)
	}
	defer s.Close()

	return start(s)
}

func serviceStop(appName string) (err error) {
	if !amAdmin() {
		return runMeElevated()
	}

	defer func() {
		l.LogError("Service Stop", errors.E(err))
		fmt.Println("Hit Enter to close")
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
	}()

	l.I("Stopping service")
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service connect failed: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(appName)
	if err != nil {
		return fmt.Errorf("open service failed: %w", err)
	}
	defer s.Close()

	return stop(s)
}

func start(s *mgr.Service) error {
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("start service failed: could not access %s: %w", s.Name, err)
	}

	if status.State == svc.StartPending || status.State == svc.PausePending || status.State == svc.StopPending || status.State == svc.ContinuePending {
		return fmt.Errorf("service in pending state")
	}
	if status.State == svc.Running {
		return nil
	}

	if status.State == svc.Stopped {
		if err = s.Start(); err != nil {
			return fmt.Errorf("start service failed %s: %w", s.Name, err)
		}
		return nil
	}

	if status.State == svc.Paused {
		if _, err = s.Control(svc.Continue); err != nil {
			return fmt.Errorf("continue service failed %s: %w", s.Name, err)
		}
	}
	return nil
}

func stop(s *mgr.Service) error {
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("could not access %s: %w", s.Name, err)
	}

	if status.State == svc.StartPending || status.State == svc.PausePending || status.State == svc.StopPending || status.State == svc.ContinuePending {
		return fmt.Errorf("service in pending state")
	}
	if status.State == svc.Stopped {
		return nil
	}

	if _, err = s.Control(svc.Stop); err != nil {
		return fmt.Errorf("could not stop service %s: %w", s.Name, err)
	}
	return nil
}

func runMeElevated() error {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 // SW_NORMAL

	return windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
}

func amAdmin() bool {
	if _, err := os.Open("\\\\.\\PHYSICALDRIVE0"); err != nil {
		return false
	}
	return true
}
