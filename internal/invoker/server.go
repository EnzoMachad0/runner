package invoker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// defaultServerStartTimeout é o tempo máximo de espera para o servidor
// ficar disponível após Start(). Pode ser reduzido em testes via export_test.go.
var defaultServerStartTimeout = 30 * time.Second

// hubsaudeDir substitui os.UserHomeDir()+"/.hubsaude" quando não vazio.
// Utilizado em testes para isolar o arquivo PID por teste.
var hubsaudeDir string

const pidFileName = "assinador.pid"

// pidInfo é a estrutura persistida no arquivo PID.
type pidInfo struct {
	PID  int `json:"pid"`
	Port int `json:"port"`
}

// Start inicia o assinador.jar em background como servidor HTTP na porta informada.
// O PID e a porta são salvos em ~/.hubsaude/assinador.pid para que IsRunning
// e Stop possam gerenciar o processo posteriormente.
//
// O jar é iniciado com: java -jar <jarPath> --server --port <port>
//
// Start não aguarda o servidor ficar disponível — use GetOrStart para isso.
// Erros possíveis:
//   - ErrJavaNotFound — javaPath não encontrado
//   - ErrJarNotFound  — jarPath não existe
//   - erros de I/O ao escrever o arquivo PID
func Start(javaPath, jarPath string, port int) error {
	// 1. Valida o executável Java.
	if _, err := exec.LookPath(javaPath); err != nil {
		return fmt.Errorf("%w: %s", ErrJavaNotFound, javaPath)
	}

	// 2. Valida a existência do JAR.
	if _, err := os.Stat(jarPath); err != nil {
		return fmt.Errorf("%w: %s", ErrJarNotFound, jarPath)
	}

	// 3. Constrói o comando no modo servidor.
	cmd := exec.Command(javaPath, "-jar", jarPath,
		"--server", "--port", strconv.Itoa(port))

	// Descarta stdout/stderr do servidor — saída vai para logs futuros (US-01.6).
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	// 4. Desvincula do processo pai para sobreviver ao encerramento do CLI.
	detachProcess(cmd)

	// 5. Inicia o processo em background.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("invoker: falha ao iniciar assinador.jar na porta %d: %w", port, err)
	}

	// 6. Persiste PID e porta para gerenciamento posterior.
	if err := writePIDFile(pidInfo{PID: cmd.Process.Pid, Port: port}); err != nil {
		// Se não conseguimos salvar o PID, mata o processo e propaga o erro.
		_ = cmd.Process.Kill()
		return err
	}

	return nil
}

// IsRunning retorna true se o assinador.jar está ativo na porta informada.
// A verificação é feita em três etapas:
//  1. Lê ~/.hubsaude/assinador.pid e confirma que a porta corresponde.
//  2. Verifica se o processo com o PID salvo ainda existe no SO.
//  3. Realiza um health check HTTP em http://localhost:<port>/health.
func IsRunning(port int) bool {
	info, err := readPIDFile()
	if err != nil {
		return false
	}

	if info.Port != port {
		return false
	}

	if !processExists(info.PID) {
		return false
	}

	return healthCheck(port)
}

// GetOrStart reutiliza a instância ativa do assinador.jar ou inicia uma nova.
// Retorna a porta onde o servidor está disponível ou um erro descritivo.
//
// Fluxo:
//  1. Se IsRunning(port) == true → retorna port imediatamente.
//  2. Caso contrário, chama Start() e aguarda o servidor ficar pronto.
//  3. Se o servidor não responder dentro de defaultServerStartTimeout, retorna erro.
func GetOrStart(javaPath, jarPath string, port int) (int, error) {
	if IsRunning(port) {
		return port, nil
	}

	if err := Start(javaPath, jarPath, port); err != nil {
		return 0, err
	}

	if !waitForReady(port, defaultServerStartTimeout) {
		return 0, fmt.Errorf(
			"invoker: assinador.jar não ficou disponível na porta %d em %v\n"+
				"Verifique se a porta está livre e se o JAR suporta o modo servidor",
			port, defaultServerStartTimeout,
		)
	}

	return port, nil
}

// --------------------------------------------------------------------------
// Helpers internos
// --------------------------------------------------------------------------

// hubsaudeBaseDir resolve o diretório ~/.hubsaude, criando-o se necessário.
// Usa hubsaudeDir quando não vazio (modo teste).
func hubsaudeBaseDir() (string, error) {
	base := hubsaudeDir
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("invoker: não foi possível determinar o diretório home: %w", err)
		}
		base = filepath.Join(home, ".hubsaude")
	}

	if err := os.MkdirAll(base, 0755); err != nil {
		return "", fmt.Errorf("invoker: não foi possível criar %s: %w", base, err)
	}

	return base, nil
}

func writePIDFile(info pidInfo) error {
	base, err := hubsaudeBaseDir()
	if err != nil {
		return err
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("invoker: falha ao serializar PID file: %w", err)
	}

	path := filepath.Join(base, pidFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("invoker: falha ao salvar PID file em %s: %w", path, err)
	}

	return nil
}

func readPIDFile() (pidInfo, error) {
	base, err := hubsaudeBaseDir()
	if err != nil {
		return pidInfo{}, err
	}

	path := filepath.Join(base, pidFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return pidInfo{}, fmt.Errorf("invoker: PID file não encontrado em %s: %w", path, err)
	}

	var info pidInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return pidInfo{}, fmt.Errorf("invoker: PID file corrompido em %s: %w", path, err)
	}

	return info, nil
}

// healthCheck faz um GET em http://localhost:<port>/health com timeout curto.
// Qualquer resposta HTTP (mesmo 4xx) indica que o servidor está de pé.
// Erro de conexão ou timeout indica que o servidor não está respondendo.
func healthCheck(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// waitForReady faz polling em healthCheck até o servidor responder ou o timeout expirar.
func waitForReady(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if healthCheck(port) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
