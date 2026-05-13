//go:build !windows

package invoker

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// processExists retorna true se o processo com o PID informado está rodando.
// Usa kill(pid, 0) do POSIX: não envia sinal, apenas verifica existência.
// EPERM indica que o processo existe mas pertence a outro usuário.
func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// detachProcess configura o comando para rodar em uma nova sessão,
// desvinculando-o do processo pai (daemoniza no sentido POSIX).
// Sem isso, o processo filho pode ser encerrado junto com o pai em alguns terminais.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}

	done := make(chan struct{})
	go func() {
		_, _ = proc.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	return nil
}
