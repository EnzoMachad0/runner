# Progresso de Implementação — Sistema Runner

**Projeto:** Trabalho prático — Implementação e Integração (UFG, 2026)
**Contexto:** SES-GO + UFG — Plataforma HubSaúde de interoperabilidade em saúde
**Última atualização:** 2026-05-05

---

## Estado geral dos sprints

| Sprint | Status | Descrição |
|--------|--------|-----------|
| Sprint 1 | Concluída (base pré-existente) | CLI base, CI/CD, GitHub Releases com Cosign |
| Sprint 2 | Parcialmente concluída | Assinatura simulada em modo local |
| Sprint 3 | Parcialmente iniciada | Modo servidor HTTP (US-01.5, US-01.7 prontas) |
| Sprint 4 | Pendente | Simulador HubSaúde |

---

## Resultado atual de `go test ./...`

```
# Módulo raiz — github.com/kyriosdata/runner
ok  github.com/kyriosdata/runner/internal/invoker   1.503s   (20 testes)
ok  github.com/kyriosdata/runner/internal/jdk       0.031s   (12 testes)

# Módulo CLI — github.com/kyriosdata/assinatura
?   github.com/kyriosdata/assinatura                [no test files]
ok  github.com/kyriosdata/assinatura/cmd            0.505s   (17 testes + subtestes)
```

**Total: 49 testes — todos passando. `go vet ./...` sem warnings em ambos os módulos.**

---

## Estrutura completa do repositório

```
runner/                                        ← raiz do repositório
├── go.mod                                     ← github.com/kyriosdata/runner, go 1.23
├── internal/
│   ├── invoker/
│   │   ├── local.go                           ← US-01.3: invocação local do jar
│   │   ├── server.go                          ← US-01.5/01.7: modo servidor HTTP
│   │   ├── platform_unix.go                   ← processExists + detachProcess (Linux/macOS)
│   │   ├── platform_windows.go                ← stubs para Windows
│   │   ├── export_test.go                     ← SetHubsaudeDir, SetServerStartTimeout
│   │   ├── local_test.go                      ← 9 testes (TestMain compartilhado)
│   │   ├── server_test.go                     ← 11 testes
│   │   └── testdata/
│   │       └── TestJar.java                   ← jar mínimo: echo, fail, sleep, server
│   └── jdk/
│       ├── detector.go                        ← US-04.1: DetectJava, DetectLocal, Get
│       └── detector_test.go                   ← 12 testes
└── projetos/
    └── assinatura/                            ← github.com/kyriosdata/assinatura
        ├── go.mod
        ├── main.go
        └── cmd/
            ├── root.go
            ├── version.go
            ├── sign.go                        ← US-01.2: 8 flags FHIR obrigatórias
            ├── validate.go                    ← US-01.2: 6 obrig. + 5 default + 5 opcionais
            └── cli_test.go                    ← 17 testes com os/exec
```

---

## Histórias implementadas

---

### US-01.2 — Comandos `sign` e `validate` no CLI

**Módulo:** `github.com/kyriosdata/assinatura` — `projetos/assinatura/cmd/`

Adiciona `assinatura sign` e `assinatura validate` ao CLI Go com Cobra.
Parsing e validação dos argumentos FHIR. A invocação do jar **não** está aqui — apenas
validação de flags. Integração com `InvokeLocal` virá na US-01.3 (integração).

#### `assinatura sign` — 8 flags, todas obrigatórias

| Flag | Tipo | Descrição |
|------|------|-----------|
| `--bundle` | string | Bundle FHIR 4.0.1 em JSON (arquivo ou conteúdo inline) |
| `--provenance` | string | Provenance FHIR 4.0.1 em JSON |
| `--cryptographic-material` | string | Chave privada/credenciais (PEM, PKCS#12, SMARTCARD, TOKEN, REMOTE) |
| `--certificates` | string | Array JSON base64 — cadeia até raiz ICP-Brasil |
| `--reference-timestamp` | int64 | Unix UTC — faixa `[1751328000, 4102444800]` |
| `--timestamp-strategy` | string | `iat` ou `tsa` |
| `--signature-policy` | string | URI `{baseUri}\|{versão}` |
| `--operational-configuration` | string | JSON de configuração operacional |

Validações no `RunE`: enum `timestamp-strategy` e faixa do `reference-timestamp`.

#### `assinatura validate` — 16 flags (6 obrig. + 5 default + 5 opcionais)

**Obrigatórias:** `--jws`, `--reference-timestamp`, `--signature-policy`, `--trust-store`,
`--revocation-policy` (`strict|soft-fail|warn`), `--ocsp-unknown-handling` (`treat-as-revoked|treat-as-warning`).

**Com default FHIR:** `--min-cert-issue-date` (1751328000), `--ocsp-crl-tsa-timeout` (30s),
`--revocation-cache-ttl` (3600s), `--near-expiry-threshold-days` (30), `--signature-age-threshold-days` (365).

**Opcionais:** `--original-bundle`, `--original-provenance`, `--max-entries-bundle`,
`--max-bundle-bytes`, `--bundle-verify-timeout`.

#### Testes (`cli_test.go`) — 17 testes com `os/exec`

`TestMain` compila o binário real. Cada teste executa o binário e verifica stdout, stderr e exit code.
Subtestes cobrem omissão de cada flag obrigatória individualmente (8 para sign, 6 para validate).

---

### US-01.3 — Invoker local (`internal/invoker/local.go`)

**Módulo:** `github.com/kyriosdata/runner`

Executa `java -jar assinador.jar <params>` como subprocesso e retorna resultado estruturado.

#### API pública

```go
func InvokeLocal(javaPath, jarPath string, params map[string]string) (string, error)
func InvokeLocalWithTimeout(javaPath, jarPath string, params map[string]string, timeout time.Duration) (string, error)
```

#### Tipos de erro

| Sentinel / Tipo | Condição |
|-----------------|----------|
| `ErrJavaNotFound` | `javaPath` inválido ou ausente no PATH |
| `ErrJarNotFound` | `jarPath` não existe |
| `*InvokeError{ExitCode, Stderr, Cause}` | exit code != 0 ou timeout |

`InvokeError` implementa `Unwrap()` para `errors.As` / `errors.Is`.

#### Fluxo

`LookPath` → `os.Stat` → monta args `--k v` → `context.WithTimeout` → `exec.CommandContext`
→ captura stdout/stderr separados → timeout tem prioridade sobre exit code.

#### Testes (`local_test.go`) — 9 testes

Integração real: `TestMain` compila `TestJar.java` com `javacFromSameJDK()` (evita
incompatibilidade de class file entre JDKs) e empacota em JAR com `archive/zip` (sem depender do `jar` do JDK).

---

### US-04.1 — Detector de JDK (`internal/jdk/detector.go`)

**Módulo:** `github.com/kyriosdata/runner`

Localiza um Java >= 21 disponível na máquina. Download automático é fora do escopo desta história.

#### API pública

```go
func DetectJava() (string, error)   // verifica java no PATH via `java -version`
func DetectLocal() (string, error)  // verifica em ~/.hubsaude/jdk/bin/java
func Get() (string, error)          // tenta DetectJava → DetectLocal → erro orientado
```

#### Comportamento de `Get()`

1. `DetectJava()` — PATH tem java >= 21? Retorna caminho.
2. `DetectLocal()` — `~/.hubsaude/jdk/bin/java` existe e é >= 21? Retorna caminho.
3. Ambos falham → erro com duas opções de correção: instalar via `https://adoptium.net`
   ou executar `assinatura jdk install`.

#### Parsing de versão

Suporta formato moderno (`"21.0.1"`) e legado (`"1.8.0_292"` → Java 8).
Regex: `version "(\d+)(?:\.(\d+))?` com tratamento do prefixo `1.X`.

#### Testabilidade

`hubsaudeHome` (var de pacote) substituível nos testes via assign direto (testes em `package jdk`).
Scripts shell falsos como executáveis em `t.TempDir()` + `t.Setenv("PATH", dir)`.

#### Testes (`detector_test.go`) — 12 testes

Cobre: `parseVersion` (7 formatos), `DetectJava` (sucesso, versão insuficiente, ausente, falha),
`DetectLocal` (sucesso, ausente, versão insuficiente), `Get` (PATH, local, prioridade PATH,
nenhum, fallback PATH-velho→local-novo).

---

### US-01.5 e US-01.7 — Modo servidor HTTP (`internal/invoker/server.go`)

**Módulo:** `github.com/kyriosdata/runner`

Gerencia o ciclo de vida do `assinador.jar` em modo servidor HTTP.
PID e porta são persistidos em `~/.hubsaude/assinador.pid` (JSON).

#### API pública

```go
func Start(javaPath, jarPath string, port int) error
func IsRunning(port int) bool
func GetOrStart(javaPath, jarPath string, port int) (int, error)
```

#### `Start()`

1. Valida java (`ErrJavaNotFound`) e jar (`ErrJarNotFound`).
2. Monta: `java -jar <jarPath> --server --port <port>`.
3. `detachProcess(cmd)` → `Setsid: true` no Linux/macOS (nova sessão POSIX); no-op no Windows.
4. `cmd.Start()` — processo em background, stdout/stderr descartados.
5. Grava `~/.hubsaude/assinador.pid` com `{"pid": N, "port": P}`.
6. Se falhar ao gravar, mata o processo e propaga o erro.

#### `IsRunning()`

Três verificações em cadeia (curto-circuito no primeiro falso):
1. Lê e desserializa `~/.hubsaude/assinador.pid` — porta coincide com a consultada?
2. `processExists(pid)` — `syscall.Kill(pid, 0)` no Unix; `os.FindProcess` no Windows.
3. `healthCheck(port)` — `GET http://localhost:<port>/health` com timeout 2s; qualquer resposta < 500 = ok.

#### `GetOrStart()`

`IsRunning` → reutiliza; ou `Start` → `waitForReady` (poll 200ms até `defaultServerStartTimeout` = 30s).

#### Arquivos de plataforma

| Arquivo | Build tag | Conteúdo |
|---------|-----------|----------|
| `platform_unix.go` | `!windows` | `processExists` via `syscall.Kill(pid, 0)` + `detachProcess` com `Setsid` |
| `platform_windows.go` | `windows` | `processExists` via `os.FindProcess` (best-effort) + `detachProcess` no-op |

#### `export_test.go`

```go
func SetHubsaudeDir(dir string) func()           // isola PID file por teste
func SetServerStartTimeout(d time.Duration) func() // acelera testes de GetOrStart
```

#### `TestJar.java` — modo servidor adicionado

`--server --port <N>`: inicia `ServerSocket` puro (sem APIs internas do JDK).
Responde `HTTP/1.1 200 OK {"status":"ok"}` a qualquer requisição.
Imprime `server:ready:<port>` ao estar pronto.

#### Testes (`server_test.go`) — 11 testes

| Grupo | Testes | Estratégia |
|-------|--------|------------|
| `TestStart_*` | EscrevePIDFile, ProcessoRodando, JavaNaoEncontrado, JarNaoEncontrado | Java real via TestJar `--server` |
| `TestIsRunning_*` | ProcessoAtivo, ProcessoMorto, PIDFileAusente, PortaDivergente, ServidorNaoResponde | `httptest.NewServer` Go + PID sintético (sem Java) |
| `TestGetOrStart_*` | ReutilizaInstanciaAtiva, IniciaNovoProcesso | Híbrido |

`TestIsRunning_ProcessoAtivo` usa `os.Getpid()` como PID — o processo de teste é o "servidor",
e um `httptest.NewServer` responde o health check. Rápido e sem JVM.

---

## O que ainda falta

### Sprint 2 — Assinatura simulada (modo local)

| História | Descrição | Status |
|----------|-----------|--------|
| US-01.3 integração | Conectar `RunE` de `sign`/`validate` ao `InvokeLocal` | Pendente |
| US-01.4 | Exibir resultado JSON do jar ao usuário | Pendente |
| US-02.1 | `assinador.jar` com modo CLI (Picocli/args) | Pendente — Java/Maven |
| US-02.2 | Validação de parâmetros no Java (fail fast) | Pendente |
| US-02.3 | Retorno JSON padronizado pelo jar | Pendente |

### Sprint 3 — Modo servidor HTTP

| História | Descrição | Status |
|----------|-----------|--------|
| US-01.5 | `Start()` — iniciar servidor | **Pronto** |
| US-01.6 | Logs do servidor | Pendente |
| US-01.7 | `IsRunning()` — health check | **Pronto** |
| US-01.8 | `Stop()` — parar servidor | Pendente |
| US-01.9 | `assinatura server start/stop/status` no CLI | Pendente |
| US-02.4 | Endpoint `/sign` no `assinador.jar` | Pendente — Java |
| US-02.5 | Endpoint `/validate` no `assinador.jar` | Pendente — Java |

### Sprint 4 — Simulador HubSaúde

| História | Descrição | Status |
|----------|-----------|--------|
| US-03.1–03.4 | CLI `simulador`, download, start/stop/status | Pendente |

---

## Decisões técnicas consolidadas

| Decisão | Racional |
|---------|----------|
| Dois módulos Go (`runner` + `assinatura`) | CLI compilável independente do runner interno; segue layout CLAUDE.md |
| `MarkFlagRequired` para flags obrigatórias | Cobra nomeia a flag ausente automaticamente; `RunE` valida apenas semântica (enum, faixa) |
| `archive/zip` para criar JAR nos testes | Sem dependência do utilitário `jar` do JDK |
| `javacFromSameJDK()` | Evita incompatibilidade de class file com múltiplos JDKs |
| `javaExec` resolvido antes de `buildTestJar` | `javacFromSameJDK` encontra o compilador correto do mesmo JDK |
| `hubsaudeDir` / `hubsaudeHome` como var de pacote | Injeção de dependência sem alterar assinatura pública das funções |
| `export_test.go` com setters de restore | Expõe internos apenas durante `go test`; não polui API de produção |
| `processExists` em arquivos de plataforma | `syscall.Kill` no Unix; stub no Windows — compila em todas as targets |
| `detachProcess` com `Setsid: true` | Servidor sobrevive ao encerramento do CLI no Linux/macOS |
| `ServerSocket` puro no TestJar | Sem APIs internas do JDK (`com.sun.*`); compatível com qualquer JVM >= 8 |
| `httptest.NewServer` nos testes de `IsRunning` | Testes rápidos sem iniciar JVM — isola a lógica de verificação |
