package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type PromptFunc func(io.Reader, io.Writer, string) (string, error)

// IsTerminal reports whether reader is an attached terminal file.
func IsTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// Secret reads a concealed terminal value or a line from a non-terminal stream.
func Secret(reader io.Reader, writer io.Writer, prompt string) (string, error) {
	if file, ok := reader.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if _, err := fmt.Fprint(writer, prompt); err != nil {
			return "", err
		}
		value, err := term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(writer)
		return strings.TrimSpace(string(value)), err
	}
	return line(reader, writer, prompt)
}

// Line writes prompt and reads one trimmed line from reader.
func Line(reader io.Reader, writer io.Writer, prompt string) (string, error) {
	return line(reader, writer, prompt)
}

func line(reader io.Reader, writer io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(writer, prompt); err != nil {
		return "", err
	}
	value, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	return strings.TrimSpace(value), nil
}
