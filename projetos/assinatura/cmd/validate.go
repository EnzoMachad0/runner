package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Flags obrigatórias do validate.
var (
	validateJWS                 string
	validateReferenceTimestamp  int64
	validateSignaturePolicy     string
	validateTrustStore          string
	validateRevocationPolicy    string
	validateOCSPUnknownHandling string
)

// Flags opcionais com valor padrão definido pela especificação FHIR.
var (
	validateMinCertIssueDate        int64
	validateOCSPCRLTSATimeout       int
	validateRevocationCacheTTL      int
	validateNearExpiryThresholdDays int
	validateSignatureAgeDays        int
)

// Flags puramente opcionais (sem valor padrão).
var (
	validateOriginalBundle      string
	validateOriginalProvenance  string
	validateMaxEntriesBundle    int
	validateMaxBundleBytes      int
	validateBundleVerifyTimeout int
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Valida uma assinatura digital JWS sobre um Bundle FHIR",
	Long: `Valida uma assinatura digital em formato JWS conforme a política
especificada, utilizando o assinador.jar para processamento local ou remoto.

Os parâmetros seguem a especificação FHIR de segurança da SES-GO/UFG:
https://fhir.saude.go.gov.br/r4/seguranca/caso-de-uso-validar-assinatura.html

Parâmetros obrigatórios: --jws, --reference-timestamp, --signature-policy,
  --trust-store, --revocation-policy, --ocsp-unknown-handling

Parâmetros com valor padrão: --min-cert-issue-date, --ocsp-crl-tsa-timeout,
  --revocation-cache-ttl, --near-expiry-threshold-days, --signature-age-threshold-days

Parâmetros opcionais: --original-bundle, --original-provenance,
  --max-entries-bundle, --max-bundle-bytes, --bundle-verify-timeout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Valida enum --revocation-policy
		switch validateRevocationPolicy {
		case "strict", "soft-fail", "warn":
			// valores válidos
		default:
			return fmt.Errorf(
				"valor inválido para --revocation-policy: %q\n"+
					"valores aceitos: strict | soft-fail | warn",
				validateRevocationPolicy,
			)
		}

		// Valida enum --ocsp-unknown-handling
		switch validateOCSPUnknownHandling {
		case "treat-as-revoked", "treat-as-warning":
			// valores válidos
		default:
			return fmt.Errorf(
				"valor inválido para --ocsp-unknown-handling: %q\n"+
					"valores aceitos: treat-as-revoked | treat-as-warning",
				validateOCSPUnknownHandling,
			)
		}

		// Valida faixa do --reference-timestamp [2025-07-01, 2100-01-01] UTC
		const tsMin int64 = 1751328000
		const tsMax int64 = 4102444800
		if validateReferenceTimestamp < tsMin || validateReferenceTimestamp > tsMax {
			return fmt.Errorf(
				"valor inválido para --reference-timestamp: %d\n"+
					"faixa válida: [%d, %d] (equivalente a 2025-07-01 até 2100-01-01 UTC)",
				validateReferenceTimestamp, tsMin, tsMax,
			)
		}

		params := map[string]string{
			"jws":                          validateJWS,
			"reference-timestamp":          fmt.Sprintf("%d", validateReferenceTimestamp),
			"signature-policy":             validateSignaturePolicy,
			"trust-store":                  validateTrustStore,
			"revocation-policy":            validateRevocationPolicy,
			"ocsp-unknown-handling":        validateOCSPUnknownHandling,
			"min-cert-issue-date":          fmt.Sprintf("%d", validateMinCertIssueDate),
			"ocsp-crl-tsa-timeout":         fmt.Sprintf("%d", validateOCSPCRLTSATimeout),
			"revocation-cache-ttl":         fmt.Sprintf("%d", validateRevocationCacheTTL),
			"near-expiry-threshold-days":   fmt.Sprintf("%d", validateNearExpiryThresholdDays),
			"signature-age-threshold-days": fmt.Sprintf("%d", validateSignatureAgeDays),
		}
		if validateOriginalBundle != "" {
			params["original-bundle"] = validateOriginalBundle
		}
		if validateOriginalProvenance != "" {
			params["original-provenance"] = validateOriginalProvenance
		}
		if validateMaxEntriesBundle > 0 {
			params["max-entries-bundle"] = fmt.Sprintf("%d", validateMaxEntriesBundle)
		}
		if validateMaxBundleBytes > 0 {
			params["max-bundle-bytes"] = fmt.Sprintf("%d", validateMaxBundleBytes)
		}
		if validateBundleVerifyTimeout > 0 {
			params["bundle-verify-timeout"] = fmt.Sprintf("%d", validateBundleVerifyTimeout)
		}

		return invokeLocalCommand("validate", params)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	f := validateCmd.Flags()
	addLocalInvokerFlags(validateCmd)

	// --- Obrigatórias ---

	f.StringVar(&validateJWS, "jws", "",
		"Assinatura digital em formato JWS JSON Serialization (base64)\n"+
			"    produzida pelo comando sign")

	f.Int64Var(&validateReferenceTimestamp, "reference-timestamp", 0,
		"Instante temporal de referência em segundos Unix UTC para todas as verificações\n"+
			"    Faixa válida: 1751328000 a 4102444800 (2025-07-01 a 2100-01-01)")

	f.StringVar(&validateSignaturePolicy, "signature-policy", "",
		"Identificador da política de assinatura no formato {baseUri}|{versão}\n"+
			"    usado para verificar conformidade e requisitos da assinatura")

	f.StringVar(&validateTrustStore, "trust-store", "",
		"Array JSON não vazio de hashes SHA-256 (hex, 64 caracteres) das\n"+
			"    autoridades certificadoras raiz ICP-Brasil aceitas")

	f.StringVar(&validateRevocationPolicy, "revocation-policy", "",
		"Política de tratamento quando o serviço de revogação está indisponível\n"+
			"    Valores: strict | soft-fail | warn")

	f.StringVar(&validateOCSPUnknownHandling, "ocsp-unknown-handling", "",
		"Tratamento do status OCSP desconhecido retornado pela AC\n"+
			"    Valores: treat-as-revoked | treat-as-warning")

	mustRequire(validateCmd, "jws")
	mustRequire(validateCmd, "reference-timestamp")
	mustRequire(validateCmd, "signature-policy")
	mustRequire(validateCmd, "trust-store")
	mustRequire(validateCmd, "revocation-policy")
	mustRequire(validateCmd, "ocsp-unknown-handling")

	// --- Opcionais com valor padrão (conforme especificação FHIR) ---

	f.Int64Var(&validateMinCertIssueDate, "min-cert-issue-date", 1751328000,
		"Data mínima para emissão de certificados em segundos Unix UTC\n"+
			"    Faixa: 1609459200 a 4102444800 | Padrão: 1751328000 (2025-07-01)")

	f.IntVar(&validateOCSPCRLTSATimeout, "ocsp-crl-tsa-timeout", 30,
		"Tempo máximo em segundos para consultas de revogação (OCSP/CRL) e TSA\n"+
			"    Faixa: 5 a 120 | Padrão: 30")

	f.IntVar(&validateRevocationCacheTTL, "revocation-cache-ttl", 3600,
		"Tempo de validade em segundos do cache de respostas de revogação\n"+
			"    Faixa: 300 a 86400 | Padrão: 3600")

	f.IntVar(&validateNearExpiryThresholdDays, "near-expiry-threshold-days", 30,
		"Limiar em dias para alertas de proximidade de expiração de certificado\n"+
			"    Faixa: 1 a 180 | Padrão: 30")

	f.IntVar(&validateSignatureAgeDays, "signature-age-threshold-days", 365,
		"Idade máxima em dias aceita para a assinatura\n"+
			"    Faixa: 1 a 1825 | Padrão: 365")

	// --- Puramente opcionais ---

	f.StringVar(&validateOriginalBundle, "original-bundle", "",
		"Bundle FHIR original para verificação de integridade do conteúdo assinado\n"+
			"    (caminho de arquivo ou conteúdo JSON; opcional)")

	f.StringVar(&validateOriginalProvenance, "original-provenance", "",
		"Provenance FHIR original para validação de ordem e histórico de assinatura\n"+
			"    (caminho de arquivo ou conteúdo JSON; opcional)")

	f.IntVar(&validateMaxEntriesBundle, "max-entries-bundle", 0,
		"Limite de segurança para o número de entradas no Bundle (0 = sem limite)")

	f.IntVar(&validateMaxBundleBytes, "max-bundle-bytes", 0,
		"Limite de segurança para o tamanho serializado do Bundle em bytes (0 = sem limite)")

	f.IntVar(&validateBundleVerifyTimeout, "bundle-verify-timeout", 0,
		"Tempo máximo em segundos para processamento da verificação (0 = sem limite)")
}
