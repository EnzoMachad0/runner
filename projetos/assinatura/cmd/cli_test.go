package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath é preenchido por TestMain com o caminho do binário compilado.
var binaryPath string

// TestMain compila o binário assinatura antes de qualquer teste.
// Todos os testes que chamam runCLI dependem desse binário.
func TestMain(m *testing.M) {
	bin, cleanup, err := compileBinary()
	if err != nil {
		// Sem o binário não há como testar; encerra com falha.
		panic("não foi possível compilar o binário de teste: " + err.Error())
	}
	binaryPath = bin
	defer cleanup()

	os.Exit(m.Run())
}

// compileBinary constrói o binário assinatura no diretório pai (raiz do módulo)
// e retorna seu caminho junto com uma função de limpeza.
func compileBinary() (path string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "assinatura-test-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	binName := "assinatura"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(tmpDir, binName)

	// O módulo raiz está em "../" relativo ao diretório cmd/.
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Join("..")
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

// runCLI executa o binário com os args informados e retorna
// stdout, stderr e o código de saída.
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

// assertContains falha o teste se s não contém substring.
func assertContains(t *testing.T, s, substring, label string) {
	t.Helper()
	if !strings.Contains(s, substring) {
		t.Errorf("%s: esperava conter %q\nConteúdo completo:\n%s", label, substring, s)
	}
}

// ============================================================
// Testes do comando sign
// ============================================================

func TestSign_Help(t *testing.T) {
	stdout, _, code := runCLI(t, "sign", "--help")
	if code != 0 {
		t.Fatalf("--help deveria retornar código 0, obteve %d", code)
	}
	for _, flag := range []string{
		"--bundle", "--provenance", "--cryptographic-material",
		"--certificates", "--reference-timestamp", "--timestamp-strategy",
		"--signature-policy", "--operational-configuration",
	} {
		assertContains(t, stdout, flag, "sign --help")
	}
}

func TestSign_SemFlags_RetornaErro(t *testing.T) {
	_, stderr, code := runCLI(t, "sign")
	if code == 0 {
		t.Fatal("esperava código de saída != 0 quando nenhuma flag é fornecida")
	}
	// Cobra lista as flags ausentes no stderr
	assertContains(t, stderr, "required flag", "stderr sign sem flags")
}

func TestSign_FlagObrigatoriaAusente(t *testing.T) {
	// Flags mínimas para sign, omitindo uma de cada vez.
	allRequired := []string{
		"--bundle", "bundle.json",
		"--provenance", "prov.json",
		"--cryptographic-material", "key.pem",
		"--certificates", "[\"base64cert\"]",
		"--reference-timestamp", "1751328000",
		"--timestamp-strategy", "iat",
		"--signature-policy", "uri|v1",
		"--operational-configuration", "{}",
	}

	cases := []struct {
		omit string // flag a omitir (sem --)
	}{
		{"bundle"},
		{"provenance"},
		{"cryptographic-material"},
		{"certificates"},
		{"reference-timestamp"},
		{"timestamp-strategy"},
		{"signature-policy"},
		{"operational-configuration"},
	}

	for _, tc := range cases {
		t.Run("omite_"+tc.omit, func(t *testing.T) {
			args := []string{"sign"}
			// Adiciona todas as flags exceto a omitida
			for i := 0; i < len(allRequired); i += 2 {
				if allRequired[i] == "--"+tc.omit {
					continue
				}
				args = append(args, allRequired[i], allRequired[i+1])
			}
			_, stderr, code := runCLI(t, args...)
			if code == 0 {
				t.Fatalf("esperava falha ao omitir --%s, mas saiu com código 0", tc.omit)
			}
			if !strings.Contains(stderr, tc.omit) {
				t.Errorf("stderr deveria mencionar %q\nstderr: %s", tc.omit, stderr)
			}
		})
	}
}

func TestSign_TodosParametros_Sucesso(t *testing.T) {
	_, _, code := runCLI(t,
		"sign",
		"--bundle", "bundle.json",
		"--provenance", "prov.json",
		"--cryptographic-material", "key.pem",
		"--certificates", `["base64cert"]`,
		"--reference-timestamp", "1751328000",
		"--timestamp-strategy", "iat",
		"--signature-policy", "https://policy.saude.go.gov.br|v1",
		"--operational-configuration", `{"trustStore":[]}`,
	)
	if code != 0 {
		t.Fatalf("esperava código 0 com todos os parâmetros, obteve %d", code)
	}
}

func TestSign_TimestampStrategyInvalido(t *testing.T) {
	_, stderr, code := runCLI(t,
		"sign",
		"--bundle", "b.json",
		"--provenance", "p.json",
		"--cryptographic-material", "k.pem",
		"--certificates", `["c"]`,
		"--reference-timestamp", "1751328000",
		"--timestamp-strategy", "invalido",
		"--signature-policy", "uri|v1",
		"--operational-configuration", "{}",
	)
	if code == 0 {
		t.Fatal("esperava falha com --timestamp-strategy inválido")
	}
	assertContains(t, stderr, "timestamp-strategy", "stderr strategy inválido")
	assertContains(t, stderr, "invalido", "stderr strategy inválido")
}

func TestSign_TimestampForaDaFaixa(t *testing.T) {
	_, stderr, code := runCLI(t,
		"sign",
		"--bundle", "b.json",
		"--provenance", "p.json",
		"--cryptographic-material", "k.pem",
		"--certificates", `["c"]`,
		"--reference-timestamp", "0",
		"--timestamp-strategy", "iat",
		"--signature-policy", "uri|v1",
		"--operational-configuration", "{}",
	)
	if code == 0 {
		t.Fatal("esperava falha com --reference-timestamp fora da faixa")
	}
	assertContains(t, stderr, "reference-timestamp", "stderr ts fora da faixa")
}

func TestSign_TimestampStrategy_TSA(t *testing.T) {
	_, _, code := runCLI(t,
		"sign",
		"--bundle", "b.json",
		"--provenance", "p.json",
		"--cryptographic-material", "k.pem",
		"--certificates", `["c"]`,
		"--reference-timestamp", "1751328000",
		"--timestamp-strategy", "tsa",
		"--signature-policy", "uri|v1",
		"--operational-configuration", "{}",
	)
	if code != 0 {
		t.Fatal("--timestamp-strategy=tsa é um valor válido e deveria ser aceito")
	}
}

// ============================================================
// Testes do comando validate
// ============================================================

func TestValidate_Help(t *testing.T) {
	stdout, _, code := runCLI(t, "validate", "--help")
	if code != 0 {
		t.Fatalf("--help deveria retornar código 0, obteve %d", code)
	}
	for _, flag := range []string{
		"--jws", "--reference-timestamp", "--signature-policy",
		"--trust-store", "--revocation-policy", "--ocsp-unknown-handling",
		"--min-cert-issue-date", "--ocsp-crl-tsa-timeout",
		"--revocation-cache-ttl", "--near-expiry-threshold-days",
		"--signature-age-threshold-days",
	} {
		assertContains(t, stdout, flag, "validate --help")
	}
}

func TestValidate_SemFlags_RetornaErro(t *testing.T) {
	_, stderr, code := runCLI(t, "validate")
	if code == 0 {
		t.Fatal("esperava código de saída != 0 quando nenhuma flag é fornecida")
	}
	assertContains(t, stderr, "required flag", "stderr validate sem flags")
}

func TestValidate_FlagObrigatoriaAusente(t *testing.T) {
	allRequired := []string{
		"--jws", "base64jws==",
		"--reference-timestamp", "1751328000",
		"--signature-policy", "uri|v1",
		"--trust-store", `["abc123"]`,
		"--revocation-policy", "strict",
		"--ocsp-unknown-handling", "treat-as-revoked",
	}

	cases := []struct{ omit string }{
		{"jws"},
		{"reference-timestamp"},
		{"signature-policy"},
		{"trust-store"},
		{"revocation-policy"},
		{"ocsp-unknown-handling"},
	}

	for _, tc := range cases {
		t.Run("omite_"+tc.omit, func(t *testing.T) {
			args := []string{"validate"}
			for i := 0; i < len(allRequired); i += 2 {
				if allRequired[i] == "--"+tc.omit {
					continue
				}
				args = append(args, allRequired[i], allRequired[i+1])
			}
			_, stderr, code := runCLI(t, args...)
			if code == 0 {
				t.Fatalf("esperava falha ao omitir --%s, mas saiu com código 0", tc.omit)
			}
			if !strings.Contains(stderr, tc.omit) {
				t.Errorf("stderr deveria mencionar %q\nstderr: %s", tc.omit, stderr)
			}
		})
	}
}

func TestValidate_TodosParametrosObrigatorios_Sucesso(t *testing.T) {
	_, _, code := runCLI(t,
		"validate",
		"--jws", "base64jws==",
		"--reference-timestamp", "1751328000",
		"--signature-policy", "https://policy.saude.go.gov.br|v1",
		"--trust-store", `["abc123"]`,
		"--revocation-policy", "strict",
		"--ocsp-unknown-handling", "treat-as-revoked",
	)
	if code != 0 {
		t.Fatalf("esperava código 0 com todos os parâmetros obrigatórios, obteve %d", code)
	}
}

func TestValidate_ComParametrosOpcionais_Sucesso(t *testing.T) {
	_, _, code := runCLI(t,
		"validate",
		"--jws", "base64jws==",
		"--reference-timestamp", "1751328000",
		"--signature-policy", "https://policy.saude.go.gov.br|v1",
		"--trust-store", `["abc123"]`,
		"--revocation-policy", "soft-fail",
		"--ocsp-unknown-handling", "treat-as-warning",
		"--min-cert-issue-date", "1609459200",
		"--ocsp-crl-tsa-timeout", "60",
		"--revocation-cache-ttl", "7200",
		"--near-expiry-threshold-days", "15",
		"--signature-age-threshold-days", "730",
		"--original-bundle", "original.json",
		"--original-provenance", "original-prov.json",
		"--max-entries-bundle", "100",
		"--max-bundle-bytes", "1048576",
		"--bundle-verify-timeout", "10",
	)
	if code != 0 {
		t.Fatalf("esperava código 0 com todos os parâmetros, obteve %d", code)
	}
}

func TestValidate_RevocationPolicyInvalida(t *testing.T) {
	_, stderr, code := runCLI(t,
		"validate",
		"--jws", "base64jws==",
		"--reference-timestamp", "1751328000",
		"--signature-policy", "uri|v1",
		"--trust-store", `["abc123"]`,
		"--revocation-policy", "invalida",
		"--ocsp-unknown-handling", "treat-as-revoked",
	)
	if code == 0 {
		t.Fatal("esperava falha com --revocation-policy inválida")
	}
	assertContains(t, stderr, "revocation-policy", "stderr revocation-policy inválida")
}

func TestValidate_OCSPUnknownHandlingInvalido(t *testing.T) {
	_, stderr, code := runCLI(t,
		"validate",
		"--jws", "base64jws==",
		"--reference-timestamp", "1751328000",
		"--signature-policy", "uri|v1",
		"--trust-store", `["abc123"]`,
		"--revocation-policy", "strict",
		"--ocsp-unknown-handling", "invalido",
	)
	if code == 0 {
		t.Fatal("esperava falha com --ocsp-unknown-handling inválido")
	}
	assertContains(t, stderr, "ocsp-unknown-handling", "stderr ocsp-unknown-handling inválido")
}

func TestValidate_TimestampForaDaFaixa(t *testing.T) {
	_, stderr, code := runCLI(t,
		"validate",
		"--jws", "base64jws==",
		"--reference-timestamp", "999",
		"--signature-policy", "uri|v1",
		"--trust-store", `["abc123"]`,
		"--revocation-policy", "strict",
		"--ocsp-unknown-handling", "treat-as-revoked",
	)
	if code == 0 {
		t.Fatal("esperava falha com --reference-timestamp fora da faixa")
	}
	assertContains(t, stderr, "reference-timestamp", "stderr ts fora da faixa")
}

// ============================================================
// Testes do comando raiz
// ============================================================

func TestRoot_Help(t *testing.T) {
	stdout, _, code := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("--help deveria retornar código 0, obteve %d", code)
	}
	assertContains(t, stdout, "sign", "root --help")
	assertContains(t, stdout, "validate", "root --help")
	assertContains(t, stdout, "version", "root --help")
}

func TestVersion(t *testing.T) {
	stdout, _, code := runCLI(t, "version")
	if code != 0 {
		t.Fatalf("version deveria retornar código 0, obteve %d", code)
	}
	assertContains(t, stdout, "Assinatura CLI", "version output")
}
