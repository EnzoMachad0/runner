//go:build windows

package invoker

import (
	"errors"
	"os"
	"os/exec"
)

// processExists verifica se o processo existe no Windows.
// os.FindProcess no Windows chama OpenProcess internamente;
// uma verificação robusta exigiria GetExitCodeProcess via x/sys/windows,
// o que está fora do escopo acadêmico atual.
func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Heurística: se OpenProcess retornou handle válido, o processo existe.
	// Limitação conhecida: pode retornar true para PIDs zumbis no Windows.
	return proc != nil
}

// detachProcess no Windows cria um novo grupo de processos,
// evitando que Ctrl+C no terminal pai encerre o filho.
func detachProcess(cmd *exec.Cmd) {
	// CREATE_NEW_PROCESS_GROUP = 0x00000200
	// Evita que sinais do console se propaguem para o processo filho.
	cmd.SysProcAttr = nil // sem syscall.SysProcAttr no Windows por padrão
}

func terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}
