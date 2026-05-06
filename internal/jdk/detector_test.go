package jdk

// Testes no mesmo pacote (package jdk) para acessar hubsaudeHome e parseVersion.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ------------------------------------------------------------------ helpers

// fakeJava cria um script shell que simula `java -version` escrevendo
// versionLine na stderr e saindo com o código informado.
// Retorna o diretório criado (já com permissão de execução no script).
// Os testes que chamam esta função são pulados automaticamente no Windows.
func fakeJava(t *testing.T, versionLine string, exitCode int) (dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripts shell não suportados no Windows; teste requer Linux/macOS")
	}

	dir = t.TempDir()
	script := "#!/bin/sh\necho '" + versionLine + "' >&2\nexit " +
		itoa(exitCode) + "\n"
	path := filepath.Join(dir, "java")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("fakeJava: %v", err)
	}
	return dir
}

// withHome substitui hubsaudeHome durante o teste e restaura ao final.
func withHome(t *testing.T, home string) {
	t.Helper()
	old := hubsaudeHome
	hubsaudeHome = home
	t.Cleanup(func() { hubsaudeHome = old })
}

// emptyDir retorna um diretório temporário vazio (sem java).
func emptyDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// itoa converte int para string sem importar strconv nos helpers.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return strings.TrimSpace(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(string(rune('0'+n)), "\x00", ""), " ", ""),
		"\n", ""))
}

// installFakeJava coloca um fake java em fakeHome/.hubsaude/jdk/bin/java.
func installFakeJava(t *testing.T, fakeHome, versionLine string, exitCode int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scripts shell não suportados no Windows")
	}
	binDir := filepath.Join(fakeHome, ".hubsaude", "jdk", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("installFakeJava MkdirAll: %v", err)
	}
	script := "#!/bin/sh\necho '" + versionLine + "' >&2\nexit " +
		itoa(exitCode) + "\n"
	path := filepath.Join(binDir, "java")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("installFakeJava WriteFile: %v", err)
	}
}

// ------------------------------------------------------------------ parseVersion

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:  "java 21 moderno",
			input: `openjdk version "21.0.1" 2023-10-17`,
			want:  21,
		},
		{
			name:  "java 17",
			input: `openjdk version "17.0.2" 2022-01-18`,
			want:  17,
		},
		{
			name:  "java 11",
			input: `openjdk version "11.0.14" 2022-01-18`,
			want:  11,
		},
		{
			name:  "java 8 formato legado 1.8",
			input: `java version "1.8.0_292" 2021-04-20`,
			want:  8,
		},
		{
			name:  "java 7 formato legado 1.7",
			input: `java version "1.7.0_80"`,
			want:  7,
		},
		{
			name:    "sem versão na saída",
			input:   "erro ao iniciar JVM",
			wantErr: true,
		},
		{
			name:    "saída vazia",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseVersion(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("esperava erro, parseVersion retornou %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tc.want {
				t.Errorf("parseVersion = %d, want %d", got, tc.want)
			}
		})
	}
}

// ------------------------------------------------------------------ DetectJava

func TestDetectJava_Sucesso(t *testing.T) {
	dir := fakeJava(t, `openjdk version "21.0.1" 2023-10-17`, 0)
	t.Setenv("PATH", dir)

	path, err := DetectJava()
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if path == "" {
		t.Fatal("caminho retornado está vazio")
	}
	if !strings.Contains(path, "java") {
		t.Errorf("caminho deveria conter 'java': %s", path)
	}
}

func TestDetectJava_VersaoInsuficiente(t *testing.T) {
	dir := fakeJava(t, `openjdk version "17.0.2" 2022-01-18`, 0)
	t.Setenv("PATH", dir)

	_, err := DetectJava()
	if err == nil {
		t.Fatal("esperava erro para versão 17, obteve nil")
	}
	if !strings.Contains(err.Error(), "17") {
		t.Errorf("erro deveria mencionar versão encontrada (17): %v", err)
	}
	if !strings.Contains(err.Error(), "21") {
		t.Errorf("erro deveria mencionar versão exigida (21): %v", err)
	}
}

func TestDetectJava_NaoEncontradoNoPATH(t *testing.T) {
	t.Setenv("PATH", emptyDir(t)) // diretório vazio, sem java

	_, err := DetectJava()
	if err == nil {
		t.Fatal("esperava erro quando java não está no PATH")
	}
	if !strings.Contains(err.Error(), "PATH") {
		t.Errorf("erro deveria mencionar PATH: %v", err)
	}
}

func TestDetectJava_FalhaAoExecutar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("scripts shell não suportados no Windows")
	}
	// Script que sai com código de erro != 0
	dir := fakeJava(t, `openjdk version "21.0.1" 2023-10-17`, 1)
	t.Setenv("PATH", dir)

	_, err := DetectJava()
	if err == nil {
		t.Fatal("esperava erro quando java -version falha")
	}
}

// ------------------------------------------------------------------ DetectLocal

func TestDetectLocal_Sucesso(t *testing.T) {
	fakeHome := t.TempDir()
	installFakeJava(t, fakeHome, `openjdk version "21.0.1" 2023-10-17`, 0)
	withHome(t, fakeHome)

	path, err := DetectLocal()
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if path == "" {
		t.Fatal("caminho retornado está vazio")
	}
	if !strings.Contains(path, ".hubsaude") {
		t.Errorf("caminho deveria conter .hubsaude: %s", path)
	}
}

func TestDetectLocal_NaoEncontrado(t *testing.T) {
	withHome(t, t.TempDir()) // home vazio, sem .hubsaude/jdk

	_, err := DetectLocal()
	if err == nil {
		t.Fatal("esperava erro quando JDK local não existe")
	}
	if !strings.Contains(err.Error(), ".hubsaude") {
		t.Errorf("erro deveria mencionar .hubsaude: %v", err)
	}
}

func TestDetectLocal_VersaoInsuficiente(t *testing.T) {
	fakeHome := t.TempDir()
	installFakeJava(t, fakeHome, `openjdk version "17.0.2" 2022-01-18`, 0)
	withHome(t, fakeHome)

	_, err := DetectLocal()
	if err == nil {
		t.Fatal("esperava erro para versão 17 no JDK local")
	}
	if !strings.Contains(err.Error(), "17") {
		t.Errorf("erro deveria mencionar a versão encontrada (17): %v", err)
	}
	if !strings.Contains(err.Error(), "21") {
		t.Errorf("erro deveria mencionar a versão exigida (21): %v", err)
	}
}

// ------------------------------------------------------------------ Get

func TestGet_JavaNoPath(t *testing.T) {
	dir := fakeJava(t, `openjdk version "21.0.1" 2023-10-17`, 0)
	t.Setenv("PATH", dir)
	withHome(t, t.TempDir()) // local vazio — não deve ser necessário

	path, err := Get()
	if err != nil {
		t.Fatalf("esperava sucesso com java no PATH, obteve: %v", err)
	}
	if path == "" {
		t.Fatal("caminho retornado está vazio")
	}
}

func TestGet_JavaLocal_QuandoPathIndisponivel(t *testing.T) {
	t.Setenv("PATH", emptyDir(t)) // sem java no PATH

	fakeHome := t.TempDir()
	installFakeJava(t, fakeHome, `openjdk version "21.0.1" 2023-10-17`, 0)
	withHome(t, fakeHome)

	path, err := Get()
	if err != nil {
		t.Fatalf("esperava sucesso com java local, obteve: %v", err)
	}
	if !strings.Contains(path, ".hubsaude") {
		t.Errorf("caminho deveria vir do JDK local (.hubsaude): %s", path)
	}
}

func TestGet_PathTemPrioridadeSobreLocal(t *testing.T) {
	// Ambas as fontes disponíveis — PATH deve ser preferida.
	pathDir := fakeJava(t, `openjdk version "21.0.1" 2023-10-17`, 0)
	t.Setenv("PATH", pathDir)

	fakeHome := t.TempDir()
	installFakeJava(t, fakeHome, `openjdk version "21.0.2" 2023-10-17`, 0)
	withHome(t, fakeHome)

	path, err := Get()
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	// Caminho retornado deve estar no diretório do PATH, não em .hubsaude
	if strings.Contains(path, ".hubsaude") {
		t.Errorf("Get() deveria preferir java do PATH, mas retornou: %s", path)
	}
}

func TestGet_NenhumJavaDisponivel(t *testing.T) {
	t.Setenv("PATH", emptyDir(t))
	withHome(t, t.TempDir())

	_, err := Get()
	if err == nil {
		t.Fatal("esperava erro quando nenhum java está disponível")
	}
	msg := err.Error()
	if !strings.Contains(msg, "21") {
		t.Errorf("erro deveria mencionar versão exigida (21): %v", err)
	}
	if !strings.Contains(msg, "PATH") {
		t.Errorf("erro deveria mencionar PATH: %v", err)
	}
	if !strings.Contains(msg, ".hubsaude") {
		t.Errorf("erro deveria mencionar .hubsaude: %v", err)
	}
}

func TestGet_PathComVersaoAntiga_LocalComVersaoCorreta(t *testing.T) {
	// PATH tem java 17 (insuficiente) → Get deve cair para local (java 21).
	oldDir := fakeJava(t, `openjdk version "17.0.2" 2022-01-18`, 0)
	t.Setenv("PATH", oldDir)

	fakeHome := t.TempDir()
	installFakeJava(t, fakeHome, `openjdk version "21.0.1" 2023-10-17`, 0)
	withHome(t, fakeHome)

	path, err := Get()
	if err != nil {
		t.Fatalf("esperava sucesso usando JDK local como fallback: %v", err)
	}
	if !strings.Contains(path, ".hubsaude") {
		t.Errorf("esperava caminho do JDK local, obteve: %s", path)
	}
}
