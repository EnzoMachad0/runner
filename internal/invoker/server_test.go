package invoker_test

// Nota: este arquivo compartilha TestMain, testJarPath, javaExec,
// skipIfNoJar e skipIfNoJava com local_test.go (mesmo package invoker_test).

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kyriosdata/runner/internal/invoker"
)

// --------------------------------------------------------------------------
// Helpers locais
// --------------------------------------------------------------------------

// freePort encontra uma porta TCP disponível no sistema.
// Há uma pequena janela de race condition entre fechar e usar; aceitável em testes.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// pidFileInDir lê e desserializa o arquivo PID do diretório informado.
func pidFileInDir(t *testing.T, dir string) struct {
	PID  int `json:"pid"`
	Port int `json:"port"`
} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "assinador.pid"))
	if err != nil {
		t.Fatalf("não foi possível ler PID file: %v", err)
	}
	var info struct {
		PID  int `json:"pid"`
		Port int `json:"port"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("PID file corrompido: %v", err)
	}
	return info
}

// inProcessHTTPServer inicia um servidor HTTP in-process usando httptest.
// Responde {"status":"ok"} em qualquer rota. Encerrado automaticamente ao fim do teste.
func inProcessHTTPServer(t *testing.T) (port int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	t.Cleanup(srv.Close)

	addr := srv.Listener.Addr().(*net.TCPAddr)
	return addr.Port
}

// writeFakePIDFile escreve um PID file sintético no dir informado.
func writeFakePIDFile(t *testing.T, dir string, pid, port int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("writeFakePIDFile MkdirAll: %v", err)
	}
	data, _ := json.Marshal(map[string]int{"pid": pid, "port": port})
	if err := os.WriteFile(filepath.Join(dir, "assinador.pid"), data, 0644); err != nil {
		t.Fatalf("writeFakePIDFile WriteFile: %v", err)
	}
}

// pollUntil chama cond repetidamente com intervalo de 100ms até retornar true
// ou o deadline expirar. Retorna o último valor de cond.
func pollUntil(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// --------------------------------------------------------------------------
// Testes de Start
// --------------------------------------------------------------------------

// TestStart_EscrevePIDFile verifica que Start() persiste PID e porta no
// arquivo ~/.hubsaude/assinador.pid após lançar o processo com sucesso.
func TestStart_EscrevePIDFile(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()
	port := freePort(t)

	if err := invoker.Start(javaExec, testJarPath, port); err != nil {
		t.Fatalf("Start: %v", err)
	}

	info := pidFileInDir(t, dir)

	// Garante que o processo foi morto ao final do teste, independentemente do resultado.
	t.Cleanup(func() {
		if proc, err := os.FindProcess(info.PID); err == nil {
			_ = proc.Kill()
		}
	})

	if info.PID <= 0 {
		t.Errorf("PID inválido no arquivo: %d", info.PID)
	}
	if info.Port != port {
		t.Errorf("porta no PID file = %d, want %d", info.Port, port)
	}
}

// TestStart_ProcessoRodando verifica que o processo iniciado por Start()
// está de fato ativo no SO após o retorno da função.
func TestStart_ProcessoRodando(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()
	port := freePort(t)

	if err := invoker.Start(javaExec, testJarPath, port); err != nil {
		t.Fatalf("Start: %v", err)
	}

	info := pidFileInDir(t, dir)
	t.Cleanup(func() {
		if proc, err := os.FindProcess(info.PID); err == nil {
			_ = proc.Kill()
		}
	})

	// Aguarda até o servidor HTTP do TestJar ficar disponível (JVM demora ~1-3s).
	ready := pollUntil(10*time.Second, func() bool {
		return invoker.IsRunning(port)
	})
	if !ready {
		t.Fatal("Start() foi chamado mas IsRunning() nunca retornou true em 10s")
	}
}

// TestStart_JavaNaoEncontrado verifica o erro quando javaPath é inválido.
func TestStart_JavaNaoEncontrado(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	err := invoker.Start("/java/inexistente/xyzzy", "/qualquer.jar", 9000)
	if err == nil {
		t.Fatal("esperava erro para java inexistente")
	}
	if !strings.Contains(err.Error(), "Java") && !strings.Contains(err.Error(), "java") {
		t.Errorf("erro deveria mencionar java: %v", err)
	}
}

// TestStart_JarNaoEncontrado verifica o erro quando jarPath não existe.
func TestStart_JarNaoEncontrado(t *testing.T) {
	skipIfNoJava(t)

	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	err := invoker.Start(javaExec, "/jar/inexistente/assinador.jar", 9000)
	if err == nil {
		t.Fatal("esperava erro para jar inexistente")
	}
	if !strings.Contains(err.Error(), "JAR") && !strings.Contains(err.Error(), "jar") {
		t.Errorf("erro deveria mencionar jar: %v", err)
	}
}

// --------------------------------------------------------------------------
// Testes de IsRunning
// --------------------------------------------------------------------------

// TestIsRunning_ProcessoAtivo usa um servidor HTTP in-process + PID file
// sintético para verificar que IsRunning retorna true sem precisar do Java.
func TestIsRunning_ProcessoAtivo(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	// Inicia servidor HTTP in-process (Go puro, sem Java).
	port := inProcessHTTPServer(t)

	// Escreve PID file apontando para este mesmo processo (os.Getpid()).
	writeFakePIDFile(t, dir, os.Getpid(), port)

	if !invoker.IsRunning(port) {
		t.Fatal("IsRunning deveria retornar true para servidor ativo com PID file correto")
	}
}

// TestIsRunning_ProcessoMorto verifica que IsRunning retorna false quando
// o PID no arquivo não corresponde a nenhum processo vivo.
func TestIsRunning_ProcessoMorto(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	// PID 9999999 quase certamente não existe em nenhum sistema.
	const pidInexistente = 9999999
	writeFakePIDFile(t, dir, pidInexistente, 9876)

	if invoker.IsRunning(9876) {
		t.Fatal("IsRunning deveria retornar false para PID inexistente")
	}
}

// TestIsRunning_PIDFileAusente verifica que IsRunning retorna false quando
// o arquivo PID não existe (nenhum servidor foi iniciado).
func TestIsRunning_PIDFileAusente(t *testing.T) {
	dir := t.TempDir() // diretório vazio — sem assinador.pid
	defer invoker.SetHubsaudeDir(dir)()

	if invoker.IsRunning(8080) {
		t.Fatal("IsRunning deveria retornar false quando PID file não existe")
	}
}

// TestIsRunning_PortaDivergente verifica que IsRunning retorna false quando
// a porta no PID file difere da porta consultada.
func TestIsRunning_PortaDivergente(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	// PID file registra porta 9000 mas consultamos 9001.
	writeFakePIDFile(t, dir, os.Getpid(), 9000)

	if invoker.IsRunning(9001) {
		t.Fatal("IsRunning deveria retornar false quando a porta diverge do PID file")
	}
}

// TestIsRunning_ServidorNaoResponde verifica que IsRunning retorna false
// quando o processo existe mas não há servidor HTTP na porta.
func TestIsRunning_ServidorNaoResponde(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	// Porta onde nenhum servidor está escutando.
	port := freePort(t)
	writeFakePIDFile(t, dir, os.Getpid(), port)

	if invoker.IsRunning(port) {
		t.Fatal("IsRunning deveria retornar false quando nenhum HTTP server está na porta")
	}
}

// --------------------------------------------------------------------------
// Testes de GetOrStart
// --------------------------------------------------------------------------

// TestGetOrStart_ReutilizaInstanciaAtiva verifica que GetOrStart retorna
// sem iniciar novo processo quando o servidor já está rodando.
func TestGetOrStart_ReutilizaInstanciaAtiva(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	port := inProcessHTTPServer(t)
	writeFakePIDFile(t, dir, os.Getpid(), port)

	got, err := invoker.GetOrStart(javaExec, "/nao/sera/usado.jar", port)
	if err != nil {
		t.Fatalf("GetOrStart: %v", err)
	}
	if got != port {
		t.Errorf("GetOrStart retornou porta %d, want %d", got, port)
	}
}

// TestGetOrStart_IniciaNovoProcesso verifica o fluxo completo: GetOrStart
// detecta que não há servidor rodando, inicia o TestJar e aguarda ele ficar pronto.
// Este é o teste de integração mais lento (aguarda JVM inicializar).
func TestGetOrStart_IniciaNovoProcesso(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()
	defer invoker.SetServerStartTimeout(15 * time.Second)()

	port := freePort(t)

	got, err := invoker.GetOrStart(javaExec, testJarPath, port)
	if err != nil {
		t.Fatalf("GetOrStart: %v", err)
	}

	// Mata o servidor ao fim do teste.
	info := pidFileInDir(t, dir)
	t.Cleanup(func() {
		if proc, err := os.FindProcess(info.PID); err == nil {
			_ = proc.Kill()
		}
	})

	if got != port {
		t.Errorf("GetOrStart retornou porta %d, want %d", got, port)
	}
	if !invoker.IsRunning(port) {
		t.Error("servidor deveria estar rodando após GetOrStart bem-sucedido")
	}
}

// --------------------------------------------------------------------------
// Testes de Stop e timeout por inatividade
// --------------------------------------------------------------------------

func TestStop_ProcessoAtivo(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()
	defer invoker.SetServerStartTimeout(15 * time.Second)()

	port := freePort(t)
	if _, err := invoker.GetOrStart(javaExec, testJarPath, port); err != nil {
		t.Fatalf("GetOrStart: %v", err)
	}

	info := pidFileInDir(t, dir)
	result, err := invoker.Stop(port)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if !result.WasRunning {
		t.Fatal("Stop deveria indicar que encerrou um processo ativo")
	}
	if result.PID != info.PID {
		t.Errorf("PID retornado = %d, want %d", result.PID, info.PID)
	}
	if _, err := os.Stat(filepath.Join(dir, "assinador.pid")); !os.IsNotExist(err) {
		t.Fatalf("PID file deveria ter sido removido, stat err=%v", err)
	}
	if processStillExists(info.PID) {
		t.Fatalf("processo %d deveria ter sido encerrado", info.PID)
	}
}

func TestStop_ProcessoJaEncerrado(t *testing.T) {
	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()

	const pidInexistente = 9999999
	writeFakePIDFile(t, dir, pidInexistente, 8080)

	result, err := invoker.Stop(8080)
	if err != nil {
		t.Fatalf("Stop deveria ser idempotente para processo já encerrado: %v", err)
	}
	if result.WasRunning {
		t.Fatal("Stop não deveria indicar processo ativo para PID inexistente")
	}
	if _, err := os.Stat(filepath.Join(dir, "assinador.pid")); !os.IsNotExist(err) {
		t.Fatalf("PID file deveria ter sido removido, stat err=%v", err)
	}
}

func TestStartWithIdleTimeout_DisparaAposInatividade(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	dir := t.TempDir()
	defer invoker.SetHubsaudeDir(dir)()
	defer invoker.SetServerStartTimeout(15 * time.Second)()

	port := freePort(t)
	if _, err := invoker.GetOrStartWithIdleTimeout(javaExec, testJarPath, port, 1500*time.Millisecond); err != nil {
		t.Fatalf("GetOrStartWithIdleTimeout: %v", err)
	}

	info := pidFileInDir(t, dir)
	stopped := pollUntil(6*time.Second, func() bool {
		_, err := os.Stat(filepath.Join(dir, "assinador.pid"))
		return os.IsNotExist(err) && !processStillExists(info.PID)
	})
	if !stopped {
		t.Fatalf("timeout por inatividade não encerrou o processo %d", info.PID)
	}
}

func processStillExists(pid int) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
