package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"

	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"

	l "launch/internal/pkg/logger"
)

func main() {
	signalQuit := make(chan os.Signal, 2)
	signal.Notify(signalQuit, os.Interrupt, os.Kill, syscall.SIGTERM)

	winsrvSignalQuit := make(chan bool) // come from Windows service cmd
	winsrvQuited := make(chan bool)     // notify Windows service that grace quit is complete
	winsrvSignalDone := make(chan bool) // come from Windows service that service is done

	log.Println("Launch Service")

	var err error
	var errMsg = ""
	defer func() {
		if rerr := recover(); rerr != nil {
			l.P("Panic occurred", l.F("panic", rerr), l.F("error", err))
		} else if err != nil {
			if errMsg == "" {
				errMsg = "Exited with error"
			}
			l.E(errMsg, l.F("error", err))
		} else {
			l.I("Finished")
		}
	}()

	// Logger Init
	if err = l.Init(true, ""); err != nil {
		errMsg = "System logger first init failed"
		return
	}

	argAppName := flag.String("appname", "", "Application name (will be used as service name)")
	argExecutable := flag.String("executable", "", "Executable file")
	argWorkdir := flag.String("workdir", "", "Working directory")
	argEnableLogFile := flag.Bool("enable-log-file", false, "Enable log to file (<workdir>/launch_log/<appname>.log)")
	argServiceInstall := flag.Bool("service-install", false, "Service install")
	argServiceUninstall := flag.Bool("service-uninstall", false, "Service uninstall")
	argServiceStart := flag.Bool("service-start", false, "Service start")
	argServiceStop := flag.Bool("service-stop", false, "Service stop")
	flag.Parse()

	if argAppName == nil || *argAppName == "" {
		err = fmt.Errorf("empty appname. -appname=myservice")
		return
	}

	if argExecutable == nil || *argExecutable == "" {
		err = fmt.Errorf("empty executable, -executable=C:/myservice/myservice.exe")
		return
	}

	if argWorkdir == nil || *argWorkdir == "" {
		err = fmt.Errorf("empty workdir, -workdir=C:/myservice")
		return
	}

	enableLogFile := false
	if argEnableLogFile != nil {
		enableLogFile = *argEnableLogFile
	}

	workDir := ""
	if argWorkdir != nil && *argWorkdir != "" {
		workDir = *argWorkdir
	}

	if *argServiceInstall {
		err = serviceInstall(*argAppName, *argExecutable, *argWorkdir, enableLogFile)
		return
	}
	if *argServiceUninstall {
		err = serviceUninstall(*argAppName)
		return
	}
	if *argServiceStart {
		err = serviceStart(*argAppName)
		return
	}
	if *argServiceStop {
		err = serviceStop(*argAppName)
		return
	}

	// Logger
	if enableLogFile {
		logFile := path.Join(workDir, "launch_log", fmt.Sprintf("%s.log", *argAppName))
		if err = l.Init(true, logFile); err != nil {
			errMsg = "System logger init failed"
			return
		}
	}

	isWinsrv := winservice(winsrvSignalQuit, winsrvQuited, winsrvSignalDone, *argAppName)

	executableFile := path.Join(*argExecutable)
	cmd := exec.Command(executableFile)
	cmd.Dir = path.Join(workDir)

	cmd.Stdout = &zapio.Writer{Log: l.Log, Level: zapcore.InfoLevel}
	cmd.Stderr = &zapio.Writer{Log: l.Log, Level: zapcore.ErrorLevel}

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("starting executable failed: %w", err)
		return
	}

	l.I("Executable started",
		l.F("appname", *argAppName),
		l.F("executable", *argExecutable),
		l.F("workdir", *argWorkdir),
		l.F("pid", cmd.Process.Pid),
	)

	select {
	case <-signalQuit:
	case <-winsrvSignalQuit:
	}

	l.I("Shutdown Service...")

	if e := cmd.Process.Kill(); e != nil {
		l.W("Shutdown executable error", l.F("err", e))
	} else {
		l.I("Shutdown executable success")
	}

	if e := cmd.Wait(); e != nil {
		l.E("Executable process finished with error", l.F("err", e))
	} else {
		l.I("Executable process finished successfully")
	}

	if isWinsrv {
		winsrvQuited <- true
		select {
		case <-winsrvSignalDone:
		case <-time.After(1 * time.Minute):
		}
	}
}
