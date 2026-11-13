package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var errAborted = errors.New("cancelled")

type prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func (p prompter) say(format string, args ...any) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

func (p prompter) ask(question, fallback string) (string, error) {
	if fallback != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", question, fallback)
	} else {
		fmt.Fprintf(p.out, "%s: ", question)
	}
	line, err := p.in.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return "", errAborted
	}
	if answer := strings.TrimSpace(line); answer != "" {
		return answer, nil
	}
	return fallback, nil
}

func (p prompter) confirm(question string, fallback bool) (bool, error) {
	hint := "y/N"
	if fallback {
		hint = "Y/n"
	}
	answer, err := p.ask(question+" ("+hint+")", "")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(answer) {
	case "":
		return fallback, nil
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

func (p prompter) required(question string) (string, error) {
	for {
		answer, err := p.ask(question, "")
		if err != nil || answer != "" {
			return answer, err
		}
	}
}

func readHidden(p prompter, question string) (string, error) {
	hide := exec.Command("stty", "-echo")
	hide.Stdin = os.Stdin
	if hide.Run() == nil {
		defer func() {
			show := exec.Command("stty", "echo")
			show.Stdin = os.Stdin
			show.Run()
			fmt.Fprintln(p.out)
		}()
	}
	return p.required(question)
}
