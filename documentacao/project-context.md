# Project Context

## Objetivo

`Northstar AI` é uma CLI em Go para preparar projetos para uso com agentes de IA e instalar artefatos operacionais de forma previsível.

O produto reduz o improviso na criação de estruturas, contexto e biblioteca local para diferentes targets de agente.

## Estado Atual do Produto

No estado atual do MVP, o produto cobre principalmente:

- inicialização do target com `northstar init <target>`
- registro de contexto canônico do projeto
- listagem de biblioteca local de assistants e skills
- instalação de artefatos da biblioteca no target ativo

O fluxo recomendado não termina no terminal. Depois do `init`, o uso principal migra para o chat com o agente instalado no target.

## Modelo Mental do Domínio

O projeto usa a seguinte linguagem ubíqua:

- `skill`: especialista com responsabilidade delimitada
- `assistant`: liderança/orquestrador da squad
- `target`: plataforma onde os artefatos serão instalados
- `northstar-squad-builder`: skill principal do produto para montagem e evolução de squads

Northstar AI não deve ser lido apenas como uma CLI de comandos. O produto existe para transformar uma necessidade de trabalho em uma squad operacional instalada no ambiente correto.

## Fluxo Principal de Uso

1. O usuário executa `northstar init <target>`.
2. O projeto recebe a estrutura base do target, como `.codex/`.
3. O agente passa a operar tools de plataforma para registrar contexto, listar biblioteca e instalar artefatos.
4. A skill principal `northstar-squad-builder` conduz a composição de squads e o reuso de assets da biblioteca.

## Targets Suportados no MVP

- `codex`
- `antigravity`
- `claude`
- `cursor`

## Fontes de Verdade

Use estas fontes nesta ordem, conforme o tipo de dúvida:

1. `README.md`: onboarding técnico, comandos e visão geral do MVP.
2. `documentacao/northstar-ai/user-docs/`: documentação de produto e uso para usuário final.
3. `src/`: implementação real, contratos e comportamento efetivo.

Se a documentação divergir do código, a inconsistência deve ser apontada explicitamente.

## Mapa do Repositório

Estrutura principal relevante:

- `src/application/`: parsing e orquestração da CLI
- `src/cmd/northstar/`: entrypoint do binário
- `src/domain/`: contratos e regras de domínio
- `src/use_case/`: casos de uso
- `src/infra/`: filesystem, catálogo, instalação e adapters
- `src/templates/default/`: biblioteca padrão de assistants, tools e templates
- `src/tests/`: testes automatizados
- `documentacao/`: documentação funcional e de produto
- `dist/`: binários gerados

## Diretriz para Novos Chats

Ao iniciar uma nova conversa neste projeto:

- leia primeiro `AGENTS.md`
- depois leia `documentacao/project-context.md`
- use `documentacao/northstar-ai/user-docs/` para contexto funcional e de produto
- confirme no `src/` o comportamento real antes de propor alteração

## Manutenção Deste Artefato

Atualize este arquivo quando mudar pelo menos um destes pontos:

- proposta de valor do produto
- linguagem ubíqua
- fluxo principal de uso
- estrutura relevante do repositório
- fontes de verdade para contexto do projeto
