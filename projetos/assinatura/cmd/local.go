package cmd

import (
	"fmt"
	"os"

	"github.com/kyriosdata/runner/internal/invoker"
	"github.com/kyriosdata/runner/internal/jdk"
	"github.com/spf13/cobra"
)

var (
	localJavaPath string
	localJarPath  string
)

func addLocalInvokerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&localJavaPath, "java", "",
		"Caminho do executável Java. Quando omitido, detecta Java >= 21 no PATH ou em ~/.hubsaude/jdk")
	cmd.Flags().StringVar(&localJarPath, "jar", "assinador.jar",
		"Caminho do assinador.jar")
}

func invokeLocalCommand(operation string, params map[string]string) error {
	javaPath, err := resolveJavaPath()
	if err != nil {
		return err
	}

	out, err := invoker.InvokeLocalCommand(javaPath, localJarPath, operation, params)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, out)
	return nil
}

func resolveJavaPath() (string, error) {
	if localJavaPath != "" {
		return localJavaPath, nil
	}
	javaPath, err := jdk.Get()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar Java compatível para executar o assinador.jar: %w", err)
	}
	return javaPath, nil
}
