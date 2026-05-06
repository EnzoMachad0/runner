// Package jdk localiza um executável Java compatível na máquina do usuário.
// A ordem de busca é: PATH → ~/.hubsaude/jdk/ (JDK provisionado localmente).
// Download automático será implementado na sequência da US-04.1.
package jdk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MinVersion é a versão mínima do Java exigida pelo assinador.jar.
const MinVersion = 21

// hubsaudeHome substitui os.UserHomeDir() quando não vazio.
// Exportado apenas para testes no mesmo módulo — não faz parte da API pública.
var hubsaudeHome string

// DetectJava verifica se existe um java >= MinVersion acessível via PATH.
// Retorna o caminho absoluto do executável em caso de sucesso.
//
// Erros possíveis:
//   - java não está no PATH
//   - java está no PATH mas a versão é inferior a MinVersion
//   - falha ao executar java -version
func DetectJava() (string, error) {
	javaPath, err := exec.LookPath("java")
	if err != nil {
		return "", fmt.Errorf(
			"java não encontrado no PATH\n"+
				"Para instalar Java %d acesse: https://adoptium.net",
			MinVersion,
		)
	}

	version, err := checkVersion(javaPath)
	if err != nil {
		return "", fmt.Errorf("java em %s: %w", javaPath, err)
	}

	if version < MinVersion {
		return "", fmt.Errorf(
			"java encontrado em %s é versão %d, mas é necessário Java %d ou superior\n"+
				"Para instalar Java %d acesse: https://adoptium.net",
			javaPath, version, MinVersion, MinVersion,
		)
	}

	return javaPath, nil
}

// DetectLocal verifica se existe um java >= MinVersion em ~/.hubsaude/jdk/bin/java.
// Esse é o caminho onde o provisionamento automático (US-04.1 completo) instala o JDK.
//
// Erros possíveis:
//   - diretório ~/.hubsaude/jdk não existe ou não contém o executável
//   - executável encontrado mas versão inferior a MinVersion
func DetectLocal() (string, error) {
	jdkDir, err := localJDKDir()
	if err != nil {
		return "", err
	}

	javaPath := filepath.Join(jdkDir, "bin", "java")
	if _, err := os.Stat(javaPath); err != nil {
		return "", fmt.Errorf(
			"JDK local não encontrado em %s\n"+
				"Execute `assinatura jdk install` para provisionar automaticamente",
			javaPath,
		)
	}

	version, err := checkVersion(javaPath)
	if err != nil {
		return "", fmt.Errorf("JDK local em %s: %w", javaPath, err)
	}

	if version < MinVersion {
		return "", fmt.Errorf(
			"JDK local em %s é versão %d, mas é necessário Java %d ou superior\n"+
				"Execute `assinatura jdk install` para provisionar a versão correta",
			javaPath, version, MinVersion,
		)
	}

	return javaPath, nil
}

// Get retorna o caminho de um java >= MinVersion disponível na máquina.
// Tenta na seguinte ordem:
//  1. DetectJava  — java no PATH
//  2. DetectLocal — java em ~/.hubsaude/jdk/
//
// Se nenhuma fonte retornar um java compatível, o erro orienta o usuário
// sobre como instalar ou provisionar o JDK.
func Get() (string, error) {
	if path, err := DetectJava(); err == nil {
		return path, nil
	}

	if path, err := DetectLocal(); err == nil {
		return path, nil
	}

	return "", fmt.Errorf(
		"Java %d ou superior não encontrado no PATH nem em ~/.hubsaude/jdk/\n"+
			"Opções para corrigir:\n"+
			"  1. Instale Java %d e adicione ao PATH: https://adoptium.net\n"+
			"  2. Execute `assinatura jdk install` para provisionar automaticamente",
		MinVersion, MinVersion,
	)
}

// checkVersion executa javaPath -version e retorna a versão major do Java.
// O comando java -version escreve na stderr por convenção histórica.
func checkVersion(javaPath string) (int, error) {
	out, err := exec.Command(javaPath, "-version").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("falha ao executar java -version: %w", err)
	}
	return parseVersion(string(out))
}

// versionRe extrai a versão do Java da saída de `java -version`.
// Suporta o formato moderno "21.0.1" (Java 9+) e o legado "1.8.0_292" (Java 8).
var versionRe = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?`)

// parseVersion extrai a versão major da saída de `java -version`.
// Exported para permitir testes unitários diretos da lógica de parsing.
func parseVersion(output string) (int, error) {
	m := versionRe.FindStringSubmatch(output)
	if m == nil {
		return 0, fmt.Errorf(
			"não foi possível identificar a versão do Java na saída: %q",
			strings.TrimSpace(output),
		)
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("versão major inválida %q: %w", m[1], err)
	}

	// Formato legado: "1.8" → Java 8, "1.7" → Java 7
	if major == 1 && m[2] != "" {
		minor, err := strconv.Atoi(m[2])
		if err != nil {
			return 0, fmt.Errorf("versão minor inválida %q: %w", m[2], err)
		}
		return minor, nil
	}

	return major, nil
}

// localJDKDir resolve o diretório do JDK provisionado (~/.hubsaude/jdk/).
// Usa hubsaudeHome se definido (somente em testes), caso contrário os.UserHomeDir().
func localJDKDir() (string, error) {
	base := hubsaudeHome
	if base == "" {
		var err error
		base, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
		}
	}
	return filepath.Join(base, ".hubsaude", "jdk"), nil
}
