package cmd

import (
	"fmt"
	"time"

	"github.com/kyriosdata/runner/internal/invoker"
	"github.com/spf13/cobra"
)

var (
	startPort           int
	startJavaPath       string
	startJarPath        string
	startTimeoutMinutes int
	stopPort            int
	statusPort          int
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Inicia o assinador.jar em modo servidor HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		if startPort <= 0 || startPort > 65535 {
			return fmt.Errorf("valor inválido para --port: %d\nInforme uma porta TCP entre 1 e 65535", startPort)
		}
		if startTimeoutMinutes < 0 {
			return fmt.Errorf("valor inválido para --timeout: %d\nInforme zero para desativar ou um número positivo de minutos", startTimeoutMinutes)
		}

		idleTimeout := time.Duration(startTimeoutMinutes) * time.Minute
		port, err := invoker.GetOrStartWithIdleTimeout(startJavaPath, startJarPath, startPort, idleTimeout)
		if err != nil {
			return err
		}

		if startTimeoutMinutes > 0 {
			cmd.Printf("Assinador iniciado na porta %d; será encerrado após %d minuto(s) sem requisições.\n", port, startTimeoutMinutes)
			return nil
		}

		cmd.Printf("Assinador iniciado na porta %d.\n", port)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Encerra o assinador.jar registrado em ~/.hubsaude/assinador.pid",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("port") && (stopPort <= 0 || stopPort > 65535) {
			return fmt.Errorf("valor inválido para --port: %d\nInforme uma porta TCP entre 1 e 65535", stopPort)
		}

		result, err := invoker.Stop(stopPort)
		if err != nil {
			return err
		}
		if result.PID == 0 {
			cmd.Println("Aviso: nenhum processo do assinador estava registrado.")
			return nil
		}
		if !result.WasRunning {
			cmd.Printf("Aviso: processo %d já estava encerrado; registro removido.\n", result.PID)
			return nil
		}

		cmd.Printf("Assinador encerrado: processo %d na porta %d.\n", result.PID, result.Port)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verifica se o assinador.jar está ativo em modo servidor HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusPort <= 0 || statusPort > 65535 {
			return fmt.Errorf("valor inválido para --port: %d\nInforme uma porta TCP entre 1 e 65535", statusPort)
		}

		if invoker.IsRunning(statusPort) {
			cmd.Printf("Assinador ativo na porta %d.\n", statusPort)
			return nil
		}

		cmd.Printf("Assinador inativo na porta %d.\n", statusPort)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)

	startCmd.Flags().IntVar(&startPort, "port", 8080, "Porta TCP do servidor HTTP")
	startCmd.Flags().StringVar(&startJavaPath, "java", "java", "Caminho do executável Java")
	startCmd.Flags().StringVar(&startJarPath, "jar", "assinador.jar", "Caminho do assinador.jar")
	startCmd.Flags().IntVar(&startTimeoutMinutes, "timeout", 0, "Minutos sem requisições antes do encerramento automático; 0 desativa")

	stopCmd.Flags().IntVar(&stopPort, "port", 0, "Porta TCP do servidor HTTP registrado")

	statusCmd.Flags().IntVar(&statusPort, "port", 8080, "Porta TCP do servidor HTTP")
}
