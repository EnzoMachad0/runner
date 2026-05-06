//go:build !windows

package invoker

import (
	"errors"
	"os/exec"
	"syscall"
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
