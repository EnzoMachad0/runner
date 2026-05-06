// export_test.go expõe variáveis internas do pacote invoker para os testes
// de integração em package invoker_test. Este arquivo é compilado APENAS
// durante `go test` e não faz parte do binário de produção.
package invoker

import "time"

// SetHubsaudeDir substitui o diretório ~/.hubsaude durante um teste e
// retorna uma função que restaura o valor original (use com defer).
//
//	restore := invoker.SetHubsaudeDir(t.TempDir())
//	defer restore()
func SetHubsaudeDir(dir string) func() {
	old := hubsaudeDir
	hubsaudeDir = dir
	return func() { hubsaudeDir = old }
}

// SetServerStartTimeout substitui o timeout de espera do GetOrStart e
// retorna uma função que restaura o valor original (use com defer).
//
//	restore := invoker.SetServerStartTimeout(5 * time.Second)
//	defer restore()
func SetServerStartTimeout(d time.Duration) func() {
	old := defaultServerStartTimeout
	defaultServerStartTimeout = d
	return func() { defaultServerStartTimeout = old }
}
