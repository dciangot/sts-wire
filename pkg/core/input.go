package core

import (
	"bufio"
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type GetInputWrapper struct {
	Scanner bufio.Reader
}

func (t *GetInputWrapper) GetPassword(question string) (password string, err error) {
	fmt.Print(question + "\n")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

func (t *GetInputWrapper) GetInputString(question string, def string) (text string, err error) {
	if def != "" {
		fmt.Print(question + "\n" + "press enter for default [" + def + "]\n")
		text, err = t.Scanner.ReadString('\n')
		if err != nil {
			return "", err
		}
		text = strings.Replace(text, "\n", "", -1)

		if text == "" {
			text = def
		}

	} else {
		fmt.Print(question + "\n")

		text, err = t.Scanner.ReadString('\n')
		if err != nil {
			return "", err
		}
		text = strings.Replace(text, "\n", "", -1)
	}

	return text, nil
}
