package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kyriosdata/runner/internal/invoker"
	"github.com/kyriosdata/runner/internal/jdk"
	"github.com/spf13/cobra"
)

const defaultPort = 8443

var (
	port    int
	jarPath string
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "simulador",
		Short: "Gerencia o simulador HubSaúde",
		Long: `CLI para iniciar, parar e consultar o simulador HubSaúde.

O simulador é exposto por padrão em http://localhost:8443.`,
	}

	root.PersistentFlags().IntVar(&port, "port", defaultPort, "Porta HTTP do simulador")
	root.PersistentFlags().StringVar(&jarPath, "jar", "simulador.jar", "Caminho do simulador.jar")

	root.AddCommand(newStartCmd(), newStopCmd(), newStatusCmd())
	return root
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Inicia o simulador.jar na porta 8443",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validatePort(port); err != nil {
				return err
			}

			javaPath, err := jdk.Get()
			if err != nil {
				return err
			}

			if _, err := doRequest(http.MethodGet, endpoint(port, "/api/info")); err == nil {
				cmd.Printf("Simulador já está respondendo na porta %d.\n", port)
				return nil
			}

			if err := invoker.Start(javaPath, jarPath, port); err != nil {
				return fmt.Errorf("não foi possível iniciar o simulador.jar na porta %d: %w", port, err)
			}

			if !waitForStatus(port, 30*time.Second) {
				return fmt.Errorf(
					"simulador.jar não ficou disponível na porta %d em 30s\n"+
						"Verifique se a porta está livre e se o JAR expõe GET /api/info",
					port,
				)
			}

			cmd.Printf("Simulador iniciado na porta %d.\n", port)
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Encerra o simulador via POST /shutdown",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validatePort(port); err != nil {
				return err
			}

			body, err := doRequest(http.MethodPost, endpoint(port, "/shutdown"))
			if err != nil {
				return fmt.Errorf("não foi possível encerrar o simulador na porta %d: %w", port, err)
			}
			if body == "" {
				cmd.Println("Simulador recebeu solicitação de encerramento.")
				return nil
			}

			cmd.Println(body)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Consulta o simulador via GET /api/info",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validatePort(port); err != nil {
				return err
			}

			body, err := doRequest(http.MethodGet, endpoint(port, "/api/info"))
			if err != nil {
				return fmt.Errorf("não foi possível consultar o simulador na porta %d: %w", port, err)
			}

			cmd.Println(body)
			return nil
		},
	}
}

func validatePort(value int) error {
	if value <= 0 || value > 65535 {
		return fmt.Errorf("valor inválido para --port: %d\nInforme uma porta TCP entre 1 e 65535", value)
	}
	return nil
}

func endpoint(port int, path string) string {
	return fmt.Sprintf("http://localhost:%d%s", port, path)
}

func doRequest(method, url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func waitForStatus(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := doRequest(http.MethodGet, endpoint(port, "/api/info")); err == nil {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
