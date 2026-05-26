// Package invoker executa o assinador.jar como subprocesso e retorna
// o resultado estruturado ao chamador.
package invoker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// DefaultTimeout é o tempo máximo de espera por uma invocação do jar.
const DefaultTimeout = 30 * time.Second

// ErrJavaNotFound é retornado quando o executável Java não pode ser localizado.
var ErrJavaNotFound = errors.New("invoker: executável Java não encontrado")

// ErrJarNotFound é retornado quando o arquivo JAR não existe no caminho informado.
var ErrJarNotFound = errors.New("invoker: arquivo JAR não encontrado")

// InvokeError representa uma falha estruturada na execução do jar.
// Permite ao chamador inspecionar ExitCode e Stderr sem parsear strings.
type InvokeError struct {
	// ExitCode é o código de saída do processo; -1 indica timeout ou sinal.
	ExitCode int
	// Stderr contém a saída de erro do processo, se houver.
	Stderr string
	// Cause encapsula o erro raiz (ex.: timeout).
	Cause error
}

func (e *InvokeError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	msg := fmt.Sprintf("invoker: processo encerrou com código %d", e.ExitCode)
	if stderr := strings.TrimSpace(e.Stderr); stderr != "" {
		msg += ": " + stderr
	}
	return msg
}

func (e *InvokeError) Unwrap() error { return e.Cause }

// InvokeLocal executa java -jar <jarPath> com os parâmetros convertidos
// em flags --chave valor. Usa DefaultTimeout como limite de tempo.
//
// javaPath pode ser um caminho absoluto (/usr/bin/java) ou um nome
// resolvível via PATH ("java"). Na implementação completa da US-04.1,
// o pacote internal/jdk será responsável por localizar ou provisionar
// o JDK quando ausente.
//
// Retorna stdout em caso de sucesso ou um dos seguintes erros:
//   - ErrJavaNotFound  — executável Java não encontrado
//   - ErrJarNotFound   — jarPath não existe
//   - *InvokeError     — processo terminou com código != 0 ou timeout
func InvokeLocal(javaPath, jarPath string, params map[string]string) (string, error) {
	return InvokeLocalWithTimeout(javaPath, jarPath, params, DefaultTimeout)
}

// InvokeLocalCommand executa java -jar <jarPath> <command> com os parâmetros
// convertidos em flags --chave valor. O command é usado pelos modos CLI do
// assinador.jar, como "sign" e "validate".
func InvokeLocalCommand(javaPath, jarPath, command string, params map[string]string) (string, error) {
	return invokeLocal(javaPath, jarPath, command, params, DefaultTimeout)
}

// InvokeLocalWithTimeout é como InvokeLocal mas aceita um timeout customizado.
// Útil em testes e em contextos onde o caller controla o prazo.
func InvokeLocalWithTimeout(javaPath, jarPath string, params map[string]string, timeout time.Duration) (string, error) {
	return invokeLocal(javaPath, jarPath, "", params, timeout)
}

func invokeLocal(javaPath, jarPath, command string, params map[string]string, timeout time.Duration) (string, error) {
	// 1. Verifica o executável Java.
	if _, err := exec.LookPath(javaPath); err != nil {
		return "", fmt.Errorf("%w: %s", ErrJavaNotFound, javaPath)
	}

	// 2. Verifica a existência do JAR.
	if _, err := os.Stat(jarPath); err != nil {
		return "", fmt.Errorf("%w: %s", ErrJarNotFound, jarPath)
	}

	// 3. Constrói os argumentos: -jar <jarPath> [--chave valor ...]
	args := []string{"-jar", jarPath}
	if command != "" {
		args = append(args, command)
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := params[k]
		args = append(args, "--"+k, v)
	}

	// 4. Executa com contexto de timeout.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, javaPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}

	// 5. Timeout tem prioridade sobre o exit code.
	if ctx.Err() != nil {
		return "", &InvokeError{
			ExitCode: -1,
			Stderr:   stderr.String(),
			Cause:    fmt.Errorf("invoker: tempo limite excedido (%v)", timeout),
		}
	}

	// 6. Processo terminou com erro.
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	return "", &InvokeError{
		ExitCode: exitCode,
		Stderr:   stderr.String(),
	}
}
