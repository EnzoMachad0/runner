# Sistema Runner

Trabalho prático da disciplina Implementação e Integração (UFG, 2026), no contexto
SES-GO + UFG e da plataforma HubSaúde de interoperabilidade em saúde.

O projeto implementa uma base para executar um `assinador.jar` Java a partir de
CLIs Go. A assinatura e a validação são simuladas: o objetivo atual é exercitar
integração, empacotamento, execução local/servidor, validação de parâmetros e
testes automatizados. Criptografia real, GUI, autenticação e persistência de
assinaturas estão fora do escopo atual.

## Estado atual

| Área | Status | O que funciona |
|------|--------|----------------|
| Sprint 1 | Concluída | CLI base, estrutura de release e artefatos assinados pré-existentes |
| Sprint 2 | Parcial | CLI `assinatura sign/validate`, invoker local, detector de JDK, modo CLI do `assinador.jar`, validação fail fast e JSON padronizado |
| Sprint 3 | Parcial | Start/stop do `assinador.jar` em modo servidor, health check e timeout de inatividade |
| Sprint 4 | Pendente | Simulador HubSaúde completo |

## Estrutura

```text
runner/
├── go.mod
├── internal/
│   ├── invoker/                 # execução local e servidor do assinador.jar
│   └── jdk/                     # detecção de Java >= 21
├── projetos/
│   ├── assinatura/              # CLI Go principal
│   └── assinador-java/          # assinador.jar Java 21
├── cmd/
│   └── simulador/               # início do CLI simulador
├── progresso.md                 # histórico detalhado do que foi implementado
├── blueprint.md                 # plano arquitetural
└── CLAUDE.md                    # contexto de trabalho do projeto
```

## Pré-requisitos

- Go instalado.
- Java 21 para o `assinador.jar`.
- Maven para rodar os testes Java via `mvn test`.

Observação: neste ambiente, os testes Go passam. O Maven não estava instalado
durante a última verificação, então o módulo Java foi validado por compilação
manual com `javac 21` e execução com Java 21.

## Como rodar os testes Go

Na raiz do repositório:

```bash
go test ./...
```

No módulo do CLI `assinatura`:

```bash
cd projetos/assinatura
go test ./...
```

O estado registrado em `progresso.md` indica todos os testes Go passando nos
módulos atuais.

## Como rodar o CLI `assinatura`

Durante desenvolvimento, use `go run` dentro do módulo:

```bash
cd projetos/assinatura
go run . --help
go run . version
```

### Comando `sign`

O comando já valida flags obrigatórias e algumas regras semânticas. Ele ainda
não chama o `assinador.jar`; essa integração está pendente na US-01.3.

```bash
go run . sign \
  --bundle '{}' \
  --provenance '{}' \
  --cryptographic-material PEM \
  --certificates '[]' \
  --reference-timestamp 1751328000 \
  --timestamp-strategy iat \
  --signature-policy 'https://policy.saude.go.gov.br|v1' \
  --operational-configuration '{}'
```

Hoje a saída esperada é uma mensagem confirmando que os parâmetros foram
validados e que a invocação do JAR será adicionada depois.

### Comando `validate`

```bash
go run . validate \
  --jws abc \
  --reference-timestamp 1751328000 \
  --signature-policy 'https://policy.saude.go.gov.br|v1' \
  --trust-store '[]' \
  --revocation-policy strict \
  --ocsp-unknown-handling treat-as-revoked
```

O comando valida:

- flags obrigatórias;
- `--reference-timestamp` na faixa de 2025-07-01 a 2100-01-01 UTC;
  - `--revocation-policy`: `strict`, `soft-fail` ou `warn`;
  - `--ocsp-unknown-handling`: `treat-as-revoked` ou `treat-as-warning`.

### Comandos de servidor

O CLI já expõe start/stop para gerenciar um `assinador.jar` em modo servidor.

```bash
go run . start --port 8080 --jar assinador.jar --java java --timeout 10
go run . stop
go run . stop --port 8080
```

O estado do processo é salvo em `~/.hubsaude/assinador.pid`. O `stop` é
idempotente: pode ser chamado mesmo se o processo já tiver terminado.

Importante: o `assinador.jar` Java real ainda não implementa servidor HTTP;
os testes Go usam um JAR de teste em `internal/invoker/testdata/TestJar.java`.

## Como rodar o `assinador.jar`

O módulo Java fica em `projetos/assinador-java`.

Com Maven instalado:

```bash
cd projetos/assinador-java
mvn test
mvn package
java -jar target/assinador-java-0.1.0-SNAPSHOT.jar --help
```

Comandos aceitos pelo JAR:

```bash
java -jar target/assinador-java-0.1.0-SNAPSHOT.jar --help

java -jar target/assinador-java-0.1.0-SNAPSHOT.jar sign \
  --bundle '{}' \
  --provenance '{}' \
  --cryptographic-material PEM \
  --certificates '[]' \
  --reference-timestamp 1751328000 \
  --timestamp-strategy iat \
  --signature-policy 'uri|v1' \
  --operational-configuration '{}'

java -jar target/assinador-java-0.1.0-SNAPSHOT.jar validate \
  --jws abc \
  --reference-timestamp 1751328000 \
  --signature-policy 'uri|v1' \
  --trust-store '[]' \
  --revocation-policy strict \
  --ocsp-unknown-handling treat-as-revoked
```

O `sign` e o `validate` retornam JSON padronizado no stdout em caso de sucesso.
Quando um parâmetro está ausente ou inválido, o JAR falha rápido, retorna código
2 e escreve um JSON padronizado no stderr.

## O que já foi implementado

### CLI Go `assinatura`

- Comando `version`.
- Comando `sign` com 8 flags obrigatórias:
  - `--bundle`
  - `--provenance`
  - `--cryptographic-material`
  - `--certificates`
  - `--reference-timestamp`
  - `--timestamp-strategy`
  - `--signature-policy`
  - `--operational-configuration`
- Comando `validate` com flags obrigatórias, flags opcionais e defaults FHIR.
- Validação de enums e faixas temporais em `sign` e `validate`.
- Testes de CLI com `os/exec`, compilando e executando o binário real.

### Invoker local

Pacote: `internal/invoker`.

Funciona para executar:

```text
java -jar assinador.jar --flag valor ...
```

APIs principais:

```go
InvokeLocal(javaPath, jarPath string, params map[string]string) (string, error)
InvokeLocalWithTimeout(javaPath, jarPath string, params map[string]string, timeout time.Duration) (string, error)
```

Tratamentos implementados:

- Java não encontrado;
- JAR inexistente;
- timeout;
- stderr e exit code preservados em erro estruturado.

### Detector de JDK

Pacote: `internal/jdk`.

Localiza Java 21 ou superior:

1. primeiro no `PATH`;
2. depois em `~/.hubsaude/jdk/bin/java`;
3. se não encontrar, retorna erro orientando instalação ou uso futuro de
   `assinatura jdk install`.

### Modo servidor no invoker

Pacote: `internal/invoker`.

APIs principais:

```go
Start(javaPath, jarPath string, port int) error
StartWithIdleTimeout(javaPath, jarPath string, port int, idleTimeout time.Duration) error
IsRunning(port int) bool
GetOrStart(javaPath, jarPath string, port int) (int, error)
GetOrStartWithIdleTimeout(javaPath, jarPath string, port int, idleTimeout time.Duration) (int, error)
Stop(port int) (StopResult, error)
```

Funciona:

- iniciar processo em background;
- persistir PID e porta em `~/.hubsaude/assinador.pid`;
- verificar processo, porta e health check;
- parar processo registrado;
- encerrar automaticamente por inatividade quando `--timeout` é informado.

### `assinador.jar` Java

Módulo: `projetos/assinador-java`.

Implementado:

- projeto Maven Java 21;
- entrada `com.kyriosdata.assinador.Main`;
- parser CLI simples para `sign` e `validate`;
- `SignatureService`;
- `FakeSignatureService`;
- `SignRequest`, `ValidateRequest` e `SignatureResponse`;
- saída JSON simulada e padronizada com `success`, `operation`, `message`,
  `data` e `errors`;
- validação fail fast para parâmetros obrigatórios, enums, faixas numéricas,
  política de assinatura e formato JSON básico;
- testes unitários para parsing básico, despacho dos comandos e cenários de
  validação inválida/erro estruturado.

## O que ainda não está pronto

- Conectar `assinatura sign` e `assinatura validate` ao `InvokeLocal`.
- Fazer o CLI Go imprimir o JSON retornado pelo JAR.
- Implementar endpoints HTTP reais `/sign` e `/validate` no `assinador.jar`.
- Implementar logs do servidor.
- Finalizar o comando `status` do servidor.
- Implementar o simulador HubSaúde completo.

## Próxima história recomendada

A próxima história é a **US-01.3 integração: conectar `sign`/`validate` do Go ao `InvokeLocal`**.

Depois dela, a sequência natural é:

1. US-01.4: exibir o JSON do JAR ao usuário.
2. US-02.4/US-02.5: implementar endpoints HTTP reais no `assinador.jar`.
