package sqlformat

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
)

var isPythonInstalled = checkPythonInstalled() == nil

var (
	ErrPythonNotInstalled   = errors.New("cmd: python not installed")
	ErrInvalidPythonVersion = errors.New("cmd: python version not valid")

	ErrPythonRequired = errors.New(`sqldump: sqlformat requires python3 to be installed. Run
  $ pip install sqlparse`)
)

func Format(stmt string) (string, error) {
	b, err := sqlformat(stmt)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// equals to $ echo 'select 1' | python3 -m sqlparse -r -
func sqlformat(stmt string) ([]byte, error) {
	r, w := io.Pipe()
	defer r.Close()

	echo := exec.Command("echo", stmt)
	sqlparse := exec.Command("python3", "-m",
		"sqlparse",
		"--reindent",           // Reindent statements
		"--indent_after_first", // Indent after first line of statement
		"--keywords", "upper",  // Change case of keywords - "upper", "lower" or "capitalize"
		"--strip-comments", // Remove comments
		"-")
	echo.Stdout = w
	sqlparse.Stdin = r
	defer w.Close()

	var stdout bytes.Buffer
	sqlparse.Stdout = &stdout

	if err := echo.Start(); err != nil {
		return nil, err
	}

	if err := sqlparse.Start(); err != nil {
		if isPythonNotFoundError(err) {
			return nil, ErrPythonRequired
		}

		return nil, err
	}

	echo.Wait()
	w.Close()
	sqlparse.Wait()
	return stdout.Bytes(), nil
}

func checkPythonInstalled() error {
	version := exec.Command("python3", "--version")
	var buf bytes.Buffer
	version.Stdout = &buf
	if err := version.Run(); err != nil {
		if isPythonNotFoundError(err) {
			return ErrPythonNotInstalled
		}

		return err
	}

	if !strings.HasPrefix(buf.String(), "Python 3") {
		return ErrInvalidPythonVersion
	}

	return nil
}

func isPythonNotFoundError(err error) bool {
	msg := err.Error()
	isPython := strings.Contains(msg, "python")
	isMissingModule := strings.Contains(msg, "executable file not found in $PATH")
	return isPython && isMissingModule
}
