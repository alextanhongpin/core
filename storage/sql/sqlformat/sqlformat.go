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
	ErrInvalidPythonVersion = errors.New("sqlformat: python version not valid")
	ErrPythonNotInstalled   = errors.New("sqlformat: python not installed")
	ErrPythonRequired       = errors.New(`sqlformat: python3 is required. Run
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
	if !isPythonInstalled {
		panic(ErrPythonRequired)
	}
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
