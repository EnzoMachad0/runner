package invoker_test

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyriosdata/runner/internal/invoker"
)

// testJarPath é preenchido pelo TestMain com o caminho do jar compilado.
// Testes que dependem do jar verificam se está vazio e pulam caso necessário.
var testJarPath string

// javaExec é o executável java resolvido no TestMain.
var javaExec string

// TestMain compila o TestJar.java e cria o jar de testes antes de rodar
// qualquer teste. Se javac não estiver disponível, os testes dependentes
// são pulados individualmente (não interrompidos aqui para não esconder
// os testes que não precisam do jar).
func TestMain(m *testing.M) {
	// javaExec deve ser resolvido antes de buildTestJar, pois
	// javacFromSameJDK o usa para localizar o compilador correto.
	java, err := exec.LookPath("java")
	if err == nil {
		javaExec = java
	}

	jar, cleanup, err := buildTestJar()
	if err != nil {
		fmt.Fprintf(os.Stderr, "aviso: jar de teste não compilado: %v\n", err)
	} else {
		testJarPath = jar
		defer cleanup()
	}

	os.Exit(m.Run())
}

// buildTestJar compila testdata/TestJar.java e empacota como jar executável.
// Usa archive/zip para criar o jar sem depender do utilitário `jar` do JDK.
func buildTestJar() (path string, cleanup func(), err error) {
	javac, err := javacFromSameJDK()
	if err != nil {
		return "", nil, err
	}

	tmpDir, err := os.MkdirTemp("", "invoker-test-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	// Compila TestJar.java → TestJar.class
	src := filepath.Join("testdata", "TestJar.java")
	out, compileErr := exec.Command(javac, "-d", tmpDir, src).CombinedOutput()
	if compileErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("compilação falhou: %w\n%s", compileErr, out)
	}

	// Empacota o .class em um JAR executável usando archive/zip
	jarPath := filepath.Join(tmpDir, "test.jar")
	if err := createExecutableJar(jarPath, tmpDir, "TestJar"); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("criação do jar falhou: %w", err)
	}

	return jarPath, cleanup, nil
}

// javacFromSameJDK localiza o compilador javac no mesmo JDK que o java em uso.
// Isso evita incompatibilidade de versão de class file quando há múltiplos
// JDKs instalados (e.g., java=17 no PATH mas javac=21 no PATH).
func javacFromSameJDK() (string, error) {
	// Tenta derivar javac do mesmo diretório que o java resolvido
	if javaExec != "" {
		resolved, err := filepath.EvalSymlinks(javaExec)
		if err == nil {
			candidate := filepath.Join(filepath.Dir(resolved), "javac")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}
	// Fallback: busca no PATH
	javac, err := exec.LookPath("javac")
	if err != nil {
		return "", fmt.Errorf("javac não encontrado (mesma JDK ou PATH): %w", err)
	}
	return javac, nil
}

// createExecutableJar monta um JAR executável (ZIP + MANIFEST.MF) em Go puro,
// sem depender do utilitário `jar` do JDK.
func createExecutableJar(jarPath, classDir, mainClass string) error {
	f, err := os.Create(jarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// MANIFEST.MF — especifica a classe principal
	manifest := fmt.Sprintf("Manifest-Version: 1.0\nMain-Class: %s\n\n", mainClass)
	mw, err := zw.Create("META-INF/MANIFEST.MF")
	if err != nil {
		return err
	}
	if _, err := mw.Write([]byte(manifest)); err != nil {
		return err
	}

	// Adiciona o arquivo .class
	classData, err := os.ReadFile(filepath.Join(classDir, mainClass+".class"))
	if err != nil {
		return err
	}
	cw, err := zw.Create(mainClass + ".class")
	if err != nil {
		return err
	}
	_, err = cw.Write(classData)
	return err
}

// skipIfNoJar pula o teste quando o jar de testes não foi compilado.
func skipIfNoJar(t *testing.T) {
	t.Helper()
	if testJarPath == "" {
		t.Skip("jar de teste não disponível (javac ausente)")
	}
}

// skipIfNoJava pula o teste quando java não está disponível.
func skipIfNoJava(t *testing.T) {
	t.Helper()
	if javaExec == "" {
		t.Skip("java não encontrado no PATH")
	}
}

// --------------------------------------------------------------------------
// Testes de sucesso
// --------------------------------------------------------------------------

func TestInvokeLocal_SemParametros(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	out, err := invoker.InvokeLocal(javaExec, testJarPath, nil)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	// Sem parâmetros, o jar não imprime nada
	if strings.TrimSpace(out) != "" {
		t.Fatalf("esperava saída vazia, obteve: %q", out)
	}
}

func TestInvokeLocal_ComParametros(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	params := map[string]string{
		"input":  "documento.xml",
		"output": "assinado.xml",
	}
	out, err := invoker.InvokeLocal(javaExec, testJarPath, params)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	// O TestJar imprime os valores que não são flags de controle
	if !strings.Contains(out, "documento.xml") {
		t.Errorf("saída não contém 'documento.xml': %q", out)
	}
	if !strings.Contains(out, "assinado.xml") {
		t.Errorf("saída não contém 'assinado.xml': %q", out)
	}
}

// --------------------------------------------------------------------------
// Testes de erro: Java não encontrado
// --------------------------------------------------------------------------

func TestInvokeLocal_JavaNaoEncontrado_CaminhoAbsoluto(t *testing.T) {
	_, err := invoker.InvokeLocal("/caminho/inexistente/java", "/qualquer/coisa.jar", nil)
	if err == nil {
		t.Fatal("esperava erro, obteve nil")
	}
	if !errors.Is(err, invoker.ErrJavaNotFound) {
		t.Fatalf("esperava ErrJavaNotFound, obteve: %v", err)
	}
}

func TestInvokeLocal_JavaNaoEncontrado_NomeInexistente(t *testing.T) {
	_, err := invoker.InvokeLocal("java-inexistente-xyzzy", "/qualquer/coisa.jar", nil)
	if err == nil {
		t.Fatal("esperava erro, obteve nil")
	}
	if !errors.Is(err, invoker.ErrJavaNotFound) {
		t.Fatalf("esperava ErrJavaNotFound, obteve: %v", err)
	}
}

// --------------------------------------------------------------------------
// Testes de erro: JAR não encontrado
// --------------------------------------------------------------------------

func TestInvokeLocal_JarNaoEncontrado(t *testing.T) {
	skipIfNoJava(t)

	_, err := invoker.InvokeLocal(javaExec, "/caminho/inexistente/assinador.jar", nil)
	if err == nil {
		t.Fatal("esperava erro, obteve nil")
	}
	if !errors.Is(err, invoker.ErrJarNotFound) {
		t.Fatalf("esperava ErrJarNotFound, obteve: %v", err)
	}
}

// --------------------------------------------------------------------------
// Testes de erro: processo termina com código != 0
// --------------------------------------------------------------------------

func TestInvokeLocal_ProcessoFalha(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	// --fail faz o TestJar sair com código 1
	params := map[string]string{"fail": "1"}
	_, err := invoker.InvokeLocal(javaExec, testJarPath, params)
	if err == nil {
		t.Fatal("esperava erro, obteve nil")
	}

	var invokeErr *invoker.InvokeError
	if !errors.As(err, &invokeErr) {
		t.Fatalf("esperava *InvokeError, obteve %T: %v", err, err)
	}
	if invokeErr.ExitCode != 1 {
		t.Errorf("esperava ExitCode=1, obteve %d", invokeErr.ExitCode)
	}
	if !strings.Contains(invokeErr.Stderr, "erro simulado") {
		t.Errorf("stderr deveria conter 'erro simulado', obteve: %q", invokeErr.Stderr)
	}
}

// --------------------------------------------------------------------------
// Testes de erro: timeout
// --------------------------------------------------------------------------

func TestInvokeLocal_Timeout(t *testing.T) {
	skipIfNoJar(t)
	skipIfNoJava(t)

	// --sleep 10000 → jar dorme 10 segundos; timeout de 200ms deve disparar antes
	params := map[string]string{"sleep": "10000"}
	_, err := invoker.InvokeLocalWithTimeout(javaExec, testJarPath, params, 200*time.Millisecond)
	if err == nil {
		t.Fatal("esperava erro de timeout, obteve nil")
	}

	var invokeErr *invoker.InvokeError
	if !errors.As(err, &invokeErr) {
		t.Fatalf("esperava *InvokeError, obteve %T: %v", err, err)
	}
	if invokeErr.ExitCode != -1 {
		t.Errorf("esperava ExitCode=-1 (timeout), obteve %d", invokeErr.ExitCode)
	}
	if !strings.Contains(invokeErr.Error(), "tempo limite") {
		t.Errorf("mensagem deveria mencionar 'tempo limite', obteve: %q", invokeErr.Error())
	}
}

// --------------------------------------------------------------------------
// Testes da mensagem de erro
// --------------------------------------------------------------------------

func TestInvokeError_Mensagem_ComCause(t *testing.T) {
	err := &invoker.InvokeError{
		ExitCode: -1,
		Cause:    fmt.Errorf("invoker: tempo limite excedido (200ms)"),
	}
	if !strings.Contains(err.Error(), "tempo limite") {
		t.Errorf("mensagem inesperada: %q", err.Error())
	}
}

func TestInvokeError_Mensagem_SemCause(t *testing.T) {
	err := &invoker.InvokeError{
		ExitCode: 2,
		Stderr:   "parâmetro inválido",
	}
	msg := err.Error()
	if !strings.Contains(msg, "código 2") {
		t.Errorf("mensagem deveria mencionar 'código 2': %q", msg)
	}
	if !strings.Contains(msg, "parâmetro inválido") {
		t.Errorf("mensagem deveria incluir stderr: %q", msg)
	}
}
