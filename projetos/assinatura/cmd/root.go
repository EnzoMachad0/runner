package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "assinatura",
	Short: "CLI para criação e validação de assinaturas digitais FHIR",
	Long: `Ferramenta de linha de comando para criar e validar assinaturas digitais
sobre recursos FHIR conforme a especificação de segurança da SES-GO/UFG (HubSaúde).`,
}

// Execute é o ponto de entrada do CLI; termina o processo em caso de erro.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
