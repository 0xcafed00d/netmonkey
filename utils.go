package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func ExitOnError(e error, exitcode int) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(exitcode)
	}
}

func PanicOnError(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		panic(e)
	}
}

// same as strings.TrimPrefix, but also returns a bool to indicate if the prefix was trimmed
func TryTrimPrefix(s, prefix string) (string, bool) {
	has := strings.HasPrefix(s, prefix)
	if has {
		return s[len(prefix):], true
	}
	return s, false
}

func ReadFile(fileName string, lineReceiver func(line string) error) error {
	f, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		err = lineReceiver(line)
		if err != nil {
			return err
		}
	}
}

type Named interface {
	Name() string
}

type Nameable interface {
	Named
	SetName(name string)
}

type NullReaderWriterCloser struct {
}

func (n NullReaderWriterCloser) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (n NullReaderWriterCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (n NullReaderWriterCloser) Close() error {
	return nil
}
