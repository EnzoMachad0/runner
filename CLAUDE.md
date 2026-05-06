# Sistema Runner — Contexto para o Claude Code

## Persona esperada

Atue como um **Engenheiro de Software Sênior** com as seguintes características:

- Qualidade em primeiro lugar: código limpo, testável e bem documentado
- Pragmatismo: soluções elegantes e viáveis dentro do contexto acadêmico
- TDD: testes escritos junto com o código de produção, nunca depois
- Segurança: considere aspectos de segurança mesmo em simulações
- Mentoria: ao explicar decisões, forneça o racional técnico para fins didáticos
- Clareza: comunicação técnica precisa; mensagens de erro devem orientar o usuário à correção

---

## Visão geral do projeto

**Sistema Runner** — trabalho prático da disciplina Implementação e Integração (UFG, 2026).
Contexto real: SES-GO e UFG desenvolvem juntos a plataforma HubSaúde de interoperabilidade em saúde.

O sistema tem três componentes principais:

| Componente      | Linguagem       | Descrição                                                                   |
| --------------- | --------------- | --------------------------------------------------------------------------- |
| `assinatura`    | Go 1.26 (Cobra) | CLI multiplataforma para criar e validar assinaturas                        |
| `simulador`     | Go 1.26 (Cobra) | CLI multiplataforma para gerenciar o simulador.jar                          |
| `assinador.jar` | Java 21 (Maven) | Simula operações de assinatura digital com validação rigorosa de parâmetros |

**O que NÃO está no escopo:** criptografia real, GUI, autenticação, armazenamento persistente de assinaturas.

---

## Stack técnica

```
Go 1.26
  - github.com/spf13/cobra v1.10.2   (CLI)
  - módulo: github.com/kyriosdata/assinatura

Java 21 / Maven
  - JUnit Jupiter 5.10.2 (testes)
  - groupId: com.kyriosdata / artifactId: assinador-java

CI/CD
  - GitHub Actions
  - Cross-compile: windows/amd64, linux/amd64, darwin/amd64
  - Cosign (Sigstore) para assinatura de artefatos
  - SemVer para versionamento (ex.: v0.1.0)
```

---

## Estrutura de diretórios

```
runner/
├── cmd/
│   ├── assinatura/          ← binário CLI principal
│   │   └── main.go
│   └── simulador/           ← binário CLI do simulador
│       └── main.go
├── internal/
│   ├── cli/                 ← parsing de comandos (cobra)
│   ├── invoker/             ← invocação do assinador.jar (local e HTTP)
│   ├── jdk/                 ← detecção e provisionamento do JDK
│   └── release/             ← download de artefatos (simulador.jar, JDK)
├── projetos/
│   ├── assinatura/          ← código Go atual do CLI
│   │   ├── cmd/
│   │   │   ├── root.go      ← comando raiz (Cobra)
│   │   │   └── version.go   ← comando `assinatura version` (já implementado)
│   │   ├── go.mod
│   │   └── main.go
│   └── assinador-java/      ← código Java atual
│       ├── pom.xml
│       └── src/main/java/com/kyriosdata/assinador/
│           ├── SignatureService.java       ← interface (já existe)
│           ├── FakeSignatureService.java   ← implementação fake (já existe)
│           └── domain/
│               ├── SignRequest.java
│               ├── ValidateRequest.java
│               └── SignatureResponse.java
├── diagramas/
│   ├── contexto.puml         ← Diagrama C4 nível 1
│   ├── conteineres.puml      ← Diagrama C4 nível 2
│   └── imagens/              ← SVGs gerados
├── docs/
│   ├── sprint-1-tasks.md     ← tarefas operacionais detalhadas
│   ├── plano-revisitado-v2.md ← plano com histórias e sprints
│   └── planejamento.md       ← orientações do professor
├── especificacao.md          ← requisitos funcionais (user stories US-01 a US-05)
├── design.md                 ← diagramas C4 e decisões arquiteturais
└── .github/
    └── workflows/
        ├── build.yml         ← CI: testes + build em push para main
        └── release.yml       ← CD: release ao criar tag v*
```

---

## O que já está implementado

- [x] `assinatura version` — exibe versão e plataforma
- [x] `SignatureService` (interface Java) — métodos `sign` e `validate`
- [x] `FakeSignatureService` (implementação Java) — retorna assinatura simulada
- [x] Estrutura Go com Cobra inicializada
- [x] `go.mod` com dependências corretas
- [x] Estrutura Maven (`pom.xml`) com JUnit Jupiter

---

## Plano de sprints

### Sprint 1 — Fundação e entrega contínua ✅ (base pronta)

Histórias: US-01.1, US-05.1, US-05.2, US-05.3
Entrega: CLI base + CI/CD + GitHub Releases com Cosign

### Sprint 2 — Assinatura simulada (modo local) ← PRÓXIMA

Histórias: US-02.1, US-02.2, US-02.3, US-01.2, US-01.3, US-01.4, US-04.1
Entrega: fluxo ponta-a-ponta: usuário executa `assinatura sign ...` e obtém resultado

### Sprint 3 — Modo servidor HTTP

Histórias: US-02.4, US-02.5, US-01.5, US-01.6, US-01.7, US-01.8, US-01.9
Entrega: assinador.jar como servidor HTTP; CLI gerencia ciclo de vida

### Sprint 4 — Simulador HubSaúde

Histórias: US-03.1, US-03.2, US-03.3, US-03.4
Entrega: sistema completo com CLI `simulador`

---

## Decisões técnicas já tomadas (DT)

| ID    | Decisão                                                     |
| ----- | ----------------------------------------------------------- |
| DT-01 | Módulo Go: `github.com/kyriosdata/runner`                   |
| DT-02 | Branch principal: `main`                                    |
| DT-03 | Plataformas: `windows/amd64`, `linux/amd64`, `darwin/amd64` |
| DT-04 | Nome dos artefatos: `assinatura-<versão>-<os>-<arch>`       |
| DT-05 | Checksums SHA256 publicados junto ao release                |
| DT-06 | Layout de pacotes conforme estrutura acima                  |

---

## Fluxo de comunicação entre componentes

```
Usuário
  └─► assinatura (CLI Go)
        ├─► [modo local]   java -jar assinador.jar <params>
        └─► [modo HTTP]    POST http://localhost:<porta>/sign
                                    └─► assinador.jar (servidor)
                                          └─► FakeSignatureService
                                                └─► SignatureResponse

Usuário
  └─► simulador (CLI Go)
        └─► simulador.jar (HTTP :8443)
              ├─► GET  /api/info     (status)
              └─► POST /shutdown     (parar)
```

---

## Referências FHIR (parâmetros de assinatura)

Os parâmetros aceitos pelo `assinador.jar` seguem as especificações:

- Criar assinatura: https://fhir.saude.go.gov.br/r4/seguranca/caso-de-uso-criar-assinatura.html
- Validar assinatura: https://fhir.saude.go.gov.br/r4/seguranca/caso-de-uso-validar-assinatura.html

---

## Convenções de código

### Go

- Erros tratados de forma explícita: nunca `_ = err`
- Pacotes em `internal/` são reutilizáveis entre `assinatura` e `simulador`
- Testes de integração usam `os/exec` para executar o binário compilado
- Mensagens de erro devem orientar o usuário: inclua o parâmetro inválido e o motivo

### Java

- Siga a estrutura de `FakeSignatureService` para novas implementações
- Valide parâmetros antes de qualquer processamento (fail fast)
- Mensagens de erro em português, claras e acionáveis
- Testes unitários com JUnit Jupiter 5 para cada cenário de validação

### Git

- Branch `main` é protegida; use branches de feature
- Commits descritivos no formato: `tipo(escopo): descrição` (ex.: `feat(cli): adiciona comando sign`)
- Pull Request antes de merge

---

## Como pedir ajuda ao Claude Code

Use prompts alinhados às histórias do plano. Exemplos:

```
# Sprint 2 — implementar comando sign
"Implemente a US-01.2: adicione o comando `sign` ao CLI Go usando Cobra.
Parâmetros conforme especificação FHIR em especificacao.md.
Inclua --help e validação de parâmetros obrigatórios."

# Sprint 2 — invocar assinador.jar localmente
"Implemente a US-01.3 em internal/invoker/local.go:
localize o java disponível (PATH ou ~/.hubsaude/jdk/),
construa e execute `java -jar assinador.jar` com os parâmetros mapeados,
capture stdout/stderr e retorne ao caller.
Trate: JDK ausente, jar não encontrado, erro de execução."

# Sprint 2 — validação de parâmetros no Java
"Implemente a US-02.2: adicione validação rigorosa de parâmetros em
FakeSignatureService.sign(). Cada campo deve ser verificado (presença e formato).
Retorne SignatureResponse com código de erro e mensagem clara para cada falha.
Escreva testes JUnit para cada cenário de validação inválida."
```

---

## Definição de pronto (DoD) geral

- Código compila sem erros (`go build ./...` / `mvn package`)
- Sem warnings de lint (`go vet ./...`)
- Testes passam (`go test ./...` / `mvn test`)
- Critérios de aceitação da história atendidos e verificáveis
- Mensagens de erro claras e orientadas à correção
- PR revisado antes do merge
