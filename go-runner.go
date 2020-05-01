/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/pkg/errors"
)

var (
	logFilePath    = flag.String("log-file", "", "If non-empty, save stdout to this file")
	alsoToStdOut   = flag.Bool("also-stdout", false, "useful with log-file, log to standard output as well as the log file")
	redirectStderr = flag.Bool("redirect-stderr", true, "treat stderr same as stdout")
)

func main() {
	flag.Parse()
	flag.Args()

	if err := configureAndRun(); err != nil {
		log.Fatal(err)
	}
}

// It will handle TERM signals gracefully and kill the process
func configureAndRun() error {
	var outputStream io.Writer
	var errStream io.Writer
	outputStream = os.Stdout
	errStream = os.Stderr

	if logFilePath != nil && *logFilePath != "" {
		logFile, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to create log file %v", *logFilePath)
		}
		if *alsoToStdOut {
			outputStream = io.MultiWriter(os.Stdout, logFile)
		} else {
			outputStream = logFile
		}
	}

	if *redirectStderr {
		errStream = outputStream
	}

	args := flag.Args()
	if len(args) == 0 {
		return errors.Errorf("not enough arguments to run")
	}
	exe := args[0]
	exeArgs := args[1:]
	cmd := exec.Command(exe, exeArgs...)
	cmd.Stdout = outputStream
	cmd.Stderr = errStream

	log.Printf("Running command:\n%v\n", cmdInfo(cmd))
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "starting command")
	}

	// Handle signals and shutdown process gracefully.
	go setupSigHandler(cmd.Process.Pid)
	return errors.Wrap(cmd.Wait(), "running command")
}

// cmdInfo generates a useful look at what the command is for printing/debug.
func cmdInfo(cmd *exec.Cmd) string {
	return fmt.Sprintf(
		`Command env: (log-file=%v, also-stdout=%v, redirect-stderr=%v)
Run from directory: %v
Executable path: %v
Args (comma-delimited): %v`, *logFilePath, *alsoToStdOut, *redirectStderr,
		cmd.Dir, cmd.Path, strings.Join(cmd.Args, ","),
	)
}

// setupSigHandler will kill the process identified by the given PID if it
// gets a TERM signal.
func setupSigHandler(pid int) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	log.Println("Now listening for interrupts")
	s := <-c
	log.Printf("Got signal: %v. Shutting down test process (PID: %v)\n", s, pid)
	p, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("Could not find process %v to shut down.\n", pid)
		return
	}
	if err := p.Signal(s); err != nil {
		log.Printf("Failed to signal test process to terminate: %v\n", err)
		return
	}
	log.Printf("Signalled process %v to terminate successfully.\n", pid)
}
