package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Variáveis de flag — capturadas pelo RunE via closure.
var (
	signBundle                string
	signProvenance            string
	signCryptographicMaterial string
	signCertificates          string
	signReferenceTimestamp    int64
	signTimestampStrategy     string
	signSignaturePolicy       string
	signOperationalConfig     string
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Cria uma assinatura digital sobre um Bundle FHIR",
	Long: `Cria uma assinatura digital sobre um Bundle FHIR conforme a política
especificada, utilizando o assinador.jar para processamento local ou remoto.

Todos os parâmetros são obrigatórios e seguem a especificação FHIR de segurança
da SES-GO/UFG: https://fhir.saude.go.gov.br/r4/seguranca/caso-de-uso-criar-assinatura.html

Os parâmetros --bundle, --provenance e --operational-configuration aceitam
caminho de arquivo (ex.: bundle.json) ou conteúdo JSON direto.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Valida enum --timestamp-strategy
		switch signTimestampStrategy {
		case "iat", "tsa":
			// valores válidos
		default:
			return fmt.Errorf(
				"valor inválido para --timestamp-strategy: %q\n"+
					"valores aceitos: iat (instante incorporado) ou tsa (autoridade de carimbo de tempo)",
				signTimestampStrategy,
			)
		}

		// Valida faixa do --reference-timestamp [2025-07-01, 2100-01-01] UTC
		const tsMin int64 = 1751328000
		const tsMax int64 = 4102444800
		if signReferenceTimestamp < tsMin || signReferenceTimestamp > tsMax {
			return fmt.Errorf(
				"valor inválido para --reference-timestamp: %d\n"+
					"faixa válida: [%d, %d] (equivalente a 2025-07-01 até 2100-01-01 UTC)",
				signReferenceTimestamp, tsMin, tsMax,
			)
		}

		// TODO (US-01.3): invocar internal/invoker.InvokeLocal com os parâmetros validados.
		cmd.Println("Parâmetros validados. Invocação do assinador.jar será adicionada na US-01.3.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(signCmd)

	f := signCmd.Flags()

	f.StringVar(&signBundle, "bundle", "",
		"Instância do recurso Bundle FHIR 4.0.1 em JSON\n"+
			"    (caminho de arquivo ou conteúdo JSON)")

	f.StringVar(&signProvenance, "provenance", "",
		"Recurso Provenance FHIR 4.0.1 com referências ao Bundle em JSON\n"+
			"    (caminho de arquivo ou conteúdo JSON)")

	f.StringVar(&signCryptographicMaterial, "cryptographic-material", "",
		"Material criptográfico para assinatura: chave privada e/ou credenciais\n"+
			"    Formatos aceitos: PEM, PKCS#12, SMARTCARD (PKCS#11), TOKEN ou REMOTE")

	f.StringVar(&signCertificates, "certificates", "",
		"Array JSON em base64 com o certificado do signatário e sua cadeia\n"+
			"    de certificação até a raiz ICP-Brasil, em ordem do signatário até a raiz")

	f.Int64Var(&signReferenceTimestamp, "reference-timestamp", 0,
		"Instante temporal de referência em segundos Unix UTC\n"+
			"    Faixa válida: 1751328000 a 4102444800 (2025-07-01 a 2100-01-01)")

	f.StringVar(&signTimestampStrategy, "timestamp-strategy", "",
		"Fonte do instante temporal da assinatura\n"+
			"    Valores: iat (instante incorporado) | tsa (autoridade de carimbo de tempo)")

	f.StringVar(&signSignaturePolicy, "signature-policy", "",
		"Identificador da política de assinatura no formato {baseUri}|{versão}\n"+
			"    conforme norma estabelecida pela SES-GO/UFG")

	f.StringVar(&signOperationalConfig, "operational-configuration", "",
		"Configuração operacional em JSON: trust store, política temporal,\n"+
			"    limites de segurança e parâmetros de middleware criptográfico\n"+
			"    (caminho de arquivo ou conteúdo JSON)")

	// Todos os parâmetros são obrigatórios para sign.
	mustRequire(signCmd, "bundle")
	mustRequire(signCmd, "provenance")
	mustRequire(signCmd, "cryptographic-material")
	mustRequire(signCmd, "certificates")
	mustRequire(signCmd, "reference-timestamp")
	mustRequire(signCmd, "timestamp-strategy")
	mustRequire(signCmd, "signature-policy")
	mustRequire(signCmd, "operational-configuration")
}

// mustRequire marca uma flag como obrigatória; entra em pânico se o nome
// não existir — indica erro de programação, não de usuário.
func mustRequire(cmd *cobra.Command, flag string) {
	if err := cmd.MarkFlagRequired(flag); err != nil {
		panic(fmt.Sprintf("flag inválida em MarkFlagRequired(%q): %v", flag, err))
	}
}
