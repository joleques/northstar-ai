# Heimdall

Heimdall é uma CLI escrita em Go para preparar projetos para uso com Assistants de IA.

O foco do produto, no estado atual do MVP, é simples:

- inicializar a estrutura de um target de agente
- registrar o contexto canônico do projeto
- listar a biblioteca de Assistants disponíveis
- instalar Assistants com suas skills associadas

Em vez de fazer o usuário montar diretórios na unha, copiar arquivo em lugar arbitrário e torcer para a IDE entender, o Heimdall organiza isso de forma previsível.


## O que o Heimdall faz

Hoje o Heimdall trabalha com a linguagem ubíqua `Assistant` como entidade principal.

Cada `Assistant`:

- tem um contrato canônico em YAML
- pode ter skills associadas
- pode ser instalado em diferentes targets

Targets suportados no MVP:

- `codex`
- `antigravity`
- `claude`
- `cursor`

## MVP atual

Interface de uso para usuário final:

- terminal: `heimdall init <target>`
- chat com Agente de IA: execução das tools de plataforma

Comandos como `start`, `list-lib`, `install` e `update-app` continuam existindo na CLI, mas no fluxo recomendado eles são acionados pelo Agente de IA através das tools de plataforma.

## Estrutura do projeto

```text
src/
├── application/       # parsers e orchestration da CLI
├── domain/            # contratos e regras de domínio
├── infra/             # catálogo e filesystem gateway
├── templates/default/ # biblioteca padrão de assistants/skills
├── tests/             # testes unitários e de integração leve
└── cmd/heimdall/      # entrypoint do binário
```

## Fluxo de uso

### 1. Inicializar um target via terminal

Exemplo com `codex`:

```bash
heimdall init codex
```

Isso prepara a estrutura básica do target no projeto.

Exemplo de diretórios gerados para `codex`:

```text
.codex/
├── assistants/
└── skills/
```

### 2. Executar o restante via chat com o Agente de IA

Depois do `init`, o fluxo recomendado é conversar com o Agente de IA para executar as tools de plataforma.

Exemplo de mensagem:

```text
$heimdall-start
Registre o contexto do projeto.
```

Tools de plataforma (categoria `platform`) para uso via chat:

- `heimdall-start`: registra contexto em `.heimdall/context` (equivalente ao comando `start`)
- `heimdall-list-lib`: lista biblioteca local de assistants/skills (equivalente ao comando `list-lib`)
- `heimdall-install`: instala artifacts da biblioteca (equivalente ao comando `install`)

Exemplos de prompt no chat (target `codex` já inicializado):

```text
$heimdall-start
Registre:
title = "Heimdall App"
description = "CLI para preparar projetos para Assistants de IA."
doc = README.md
```

```text
$heimdall-list-lib
Mostre assistants e skills da categoria documentation.
```

```text
$heimdall-install
Instale os itens da categoria documentation.
```

Fluxo sugerido no chat:

1. Pedir `heimdall-start` para criar/atualizar o contexto do projeto.
2. Pedir `heimdall-list-lib` para descobrir o catálogo disponível.
3. Pedir `heimdall-install` para instalar por id, por categoria ou instalação geral.

Observação: o `target` usado nas tools vem do contexto salvo no `init` (ex.: `codex`), sem precisar repetir isso manualmente a cada execução.

## Comando de CLI para usuário final

```bash
heimdall init <codex|antigravity|claude|cursor> \
  [--agents-policy <skip|if-missing|overwrite>] \
  [--force] \
  [--output <dir>]
```

## Como rodar localmente

Executando sem gerar binário:

```bash
go run ./src/cmd/heimdall
```

Exemplo real:

```bash
go run ./src/cmd/heimdall init codex
```

## Como gerar o executável

O entrypoint do projeto está em `./src/cmd/heimdall`.

### Build local para o sistema atual

```bash
go build -o heimdall ./src/cmd/heimdall
```

No Windows:

```bash
go build -o heimdall.exe ./src/cmd/heimdall
```

### Gerar executáveis para Linux, Windows e macOS

Crie uma pasta de saída:

```bash
mkdir -p dist
```

#### Linux amd64

```bash
GOOS=linux GOARCH=amd64 go build -o dist/heimdall-linux-amd64 ./src/cmd/heimdall
```

#### Linux arm64

```bash
GOOS=linux GOARCH=arm64 go build -o dist/heimdall-linux-arm64 ./src/cmd/heimdall
```

#### Windows amd64

```bash
GOOS=windows GOARCH=amd64 go build -o dist/heimdall-windows-amd64.exe ./src/cmd/heimdall
```

#### Windows arm64

```bash
GOOS=windows GOARCH=arm64 go build -o dist/heimdall-windows-arm64.exe ./src/cmd/heimdall
```

#### macOS amd64

```bash
GOOS=darwin GOARCH=amd64 go build -o dist/heimdall-darwin-amd64 ./src/cmd/heimdall
```

#### macOS arm64

```bash
GOOS=darwin GOARCH=arm64 go build -o dist/heimdall-darwin-arm64 ./src/cmd/heimdall
```

### Gerar todos de uma vez

Linux/macOS shell:

```bash
mkdir -p dist

GOOS=linux GOARCH=amd64 go build -o dist/heimdall-linux-amd64 ./src/cmd/heimdall
GOOS=linux GOARCH=arm64 go build -o dist/heimdall-linux-arm64 ./src/cmd/heimdall
GOOS=windows GOARCH=amd64 go build -o dist/heimdall-windows-amd64.exe ./src/cmd/heimdall
GOOS=windows GOARCH=arm64 go build -o dist/heimdall-windows-arm64.exe ./src/cmd/heimdall
GOOS=darwin GOARCH=amd64 go build -o dist/heimdall-darwin-amd64 ./src/cmd/heimdall
GOOS=darwin GOARCH=arm64 go build -o dist/heimdall-darwin-arm64 ./src/cmd/heimdall
```

## Como validar antes de testar

```bash
go test ./...
go build ./...
```

## Status do produto

Entrega 1 do MVP está funcional:

- `init`
- `start`
- `list-lib`
- `install`
- `update-app`

Próximo ciclo natural:

- consolidar documentação de uso
- evoluir criação, edição e remoção de Assistants
- ampliar a biblioteca padrão

## Licença

Este projeto está disponível como código aberto para estudo, evolução e uso pela comunidade.
