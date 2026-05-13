package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	bin, cleanup, err := compileBinary()
	if err != nil {
		panic("não foi possível compilar o binário de teste: " + err.Error())
	}
	binaryPath = bin
	defer cleanup()

	os.Exit(m.Run())
}

func compileBinary() (path string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "simulador-test-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	binName := "simulador"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(tmpDir, binName)

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, &buildError{err: err, output: string(out)}
	}

	return binPath, cleanup, nil
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string {
	return "build falhou: " + e.err.Error() + "\n" + e.output
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return
}

func assertContains(t *testing.T, value, substring, label string) {
	t.Helper()
	if !strings.Contains(value, substring) {
		t.Errorf("%s: esperava conter %q\nConteúdo completo:\n%s", label, substring, value)
	}
}

func TestHelpFunciona(t *testing.T) {
	stdout, _, code := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("--help deveria retornar código 0, obteve %d", code)
	}

	assertContains(t, stdout, "simulador", "simulador --help")
	assertContains(t, stdout, "start", "simulador --help")
	assertContains(t, stdout, "stop", "simulador --help")
	assertContains(t, stdout, "status", "simulador --help")
}

func TestSubcomandosPossuemHelpProprio(t *testing.T) {
	for _, subcommand := range []string{"start", "stop", "status"} {
		t.Run(subcommand, func(t *testing.T) {
			stdout, _, code := runCLI(t, subcommand, "--help")
			if code != 0 {
				t.Fatalf("%s --help deveria retornar código 0, obteve %d", subcommand, code)
			}

			assertContains(t, stdout, subcommand, subcommand+" --help")
			assertContains(t, stdout, "--port", subcommand+" --help")
		})
	}
}
