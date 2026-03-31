# Sistema Runner — Blueprint de Implementação

## Stack proposta

### 1. Aplicação `assinatura` (CLI multiplataforma)

- **Linguagem:** Go
- **CLI:** Cobra
- **HTTP client:** `net/http`
- **Execução de processos:** `os/exec`
- **Configuração local:** JSON ou YAML em diretório do usuário
- **Empacotamento:** GoReleaser

### 2. Aplicação `assinador.jar`

- **Linguagem:** Java 21
- **CLI parsing:** Picocli
- **Servidor HTTP embutido:** Spring Boot Web ou Javalin
- **Build:** Maven ou Gradle
- **Saída:** JSON estruturado para integração com o CLI

## Decisão arquitetural

A divisão mais segura e simples para o trabalho é:

- **Go** para o executável final multiplataforma.
- **Java** para o `assinador.jar`, conforme exigido no enunciado.

Isso atende diretamente aos requisitos de:

- CLI multiplataforma
- execução de `jar`
- distribuição em GitHub Releases
- facilidade de automação no CI/CD

## Módulos do CLI `assinatura`

```text
assinatura/
├── cmd/
│   ├── root.go
│   ├── sign.go
│   ├── verify.go
│   ├── server.go
│   ├── simulator.go
│   └── jdk.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── runner/
│   ├── signer/
│   ├── simulator/
│   ├── jdk/
│   ├── release/
│   └── output/
├── pkg/
│   ├── fsutil/
│   ├── httpx/
│   └── platform/
├── main.go
└── go.mod
```

## Módulos do `assinador.jar`

```text
assinador/
├── src/main/java/br/ufg/runner/
│   ├── cli/
│   ├── http/
│   ├── service/
│   ├── validation/
│   ├── model/
│   └── exception/
├── src/main/resources/
├── pom.xml
└── README.md
```

## Casos de uso principais

### US-01 — Invocar `assinador.jar`

O CLI deve suportar dois modos:

- **local**: `java -jar assinador.jar sign ...`
- **server**: `POST /sign` e `POST /verify`

Estratégia padrão:

1. verificar se existe servidor ativo na porta padrão;
2. se existir, usar HTTP;
3. se não existir, iniciar servidor e usar HTTP;
4. se o usuário passar `--local`, executar diretamente.

### US-02 — Simular assinatura e validação

O `assinador.jar` deve:

- validar todos os parâmetros recebidos;
- retornar JSON padronizado;
- simular assinatura com payload fixo;
- simular validação com regra simples.

### US-03 — Gerenciar simulador

O CLI deve:

- baixar a release mais recente do `simulador.jar`;
- verificar se já existe localmente;
- iniciar/parar/verificar status;
- validar portas antes de iniciar.

### US-04 — Provisionar JDK

O CLI deve:

- detectar se existe Java compatível;
- baixar JDK se necessário;
- usar o JDK local da aplicação quando ausente no sistema.

### US-05 — Distribuição multiplataforma

Usar GoReleaser para gerar:

- Windows `.exe`
- Linux binário ou AppImage
- macOS binário
- checksums SHA256
- assinatura com Cosign

## Contrato JSON sugerido

### Sucesso — assinatura

```json
{
    "success": true,
    "operation": "sign",
    "message": "Assinatura simulada gerada com sucesso.",
    "data": {
        "signature": "SIMULATED_BASE64_SIGNATURE",
        "algorithm": "SHA256withRSA",
        "signedAt": "2026-03-31T12:00:00Z"
    }
}
```

### Sucesso — validação

```json
{
    "success": true,
    "operation": "verify",
    "message": "Assinatura válida.",
    "data": {
        "valid": true,
        "signer": "Certificado Simulado",
        "checkedAt": "2026-03-31T12:00:00Z"
    }
}
```

### Erro

```json
{
    "success": false,
    "operation": "sign",
    "message": "Parâmetro obrigatório ausente: --input",
    "errors": [
        {
            "field": "input",
            "reason": "required"
        }
    ]
}
```

## Comandos do CLI sugeridos

```bash
assinatura sign --input documento.xml --output assinatura.xml
assinatura verify --input assinatura.xml
assinatura server start --port 8080
assinatura server stop --port 8080
assinatura server status --port 8080
assinatura simulator start
assinatura simulator stop
assinatura simulator status
assinatura jdk install
```

## Critérios de implementação por fase

### Fase 1 — MVP mínimo

- CLI em Go com comandos `sign` e `verify`
- `assinador.jar` com modo local
- validação de parâmetros
- retorno JSON
- testes unitários básicos

### Fase 2 — modo servidor

- endpoints HTTP no `assinador.jar`
- detecção de instância já ativa
- `server start/stop/status`
- timeout e shutdown programado

### Fase 3 — simulador e JDK

- download do `simulador.jar` por releases
- gerenciamento de processo
- provisionamento automático de JDK

### Fase 4 — release e supply chain

- GitHub Actions
- GoReleaser
- checksums
- Cosign
- documentação final

## Estrutura de testes

### CLI Go

- parsing de comandos
- seleção de modo local vs servidor
- tratamento de erro
- status de processo
- download de artefatos

### Java

- validação de parâmetros
- respostas simuladas
- endpoints HTTP
- tratamento de exceções

### Integração

- CLI chamando `jar` localmente
- CLI chamando `jar` via HTTP
- inicialização automática do servidor
- erro de porta ocupada

## Próximo passo recomendado

Começar pelo **MVP da Fase 1**:

1. criar CLI `assinatura` em Go;
2. criar `assinador.jar` em Java com comando `sign` e `verify`;
3. padronizar JSON de saída;
4. escrever testes mínimos.

## Entrega acadêmica sugerida

Para o trabalho, vocês podem apresentar:

- arquitetura
- decisões técnicas
- backlog por histórias de usuário
- estrutura dos repositórios
- MVP funcionando
- pipeline de release com assinatura
