---
name: triagem-demanda
description: Valida a classificacao obrigatoria da demanda no inicio do chat, bloqueia continuidade sem tipo explicito e aplica o protocolo correto para consulta, bug, melhoria, evolucao ou nova funcionalidade.
---

# Triagem de Demanda

Use esta skill quando a conversa estiver no inicio ou quando a natureza da demanda mudar durante o chat.

## Objetivo

Garantir que a demanda esteja classificada corretamente antes de qualquer analise aprofundada, plano de implementacao ou alteracao de arquivos.

## Tipos aceitos

- `consulta`
- `bug`
- `melhoria`
- `evolucao`
- `nova funcionalidade`

## Regras obrigatorias

- A classificacao deve ser informada explicitamente pelo usuario.
- Sem classificacao, o agente deve interromper o fluxo e solicitar a classificacao.
- Se a demanda mudar de natureza no meio da conversa, o agente deve solicitar reclassificacao antes de seguir.
- Em `consulta`, o agente pode analisar documentacao e codigo, mas nao pode alterar arquivos de implementacao, testes ou configuracao.
- Em `bug`, `melhoria`, `evolucao` e `nova funcionalidade`, o agente deve seguir para plano obrigatorio antes de implementar.

## Fluxo

1. Verifique se o usuario informou um dos tipos aceitos.
2. Se nao informou, solicite a classificacao e nao avance.
3. Se informou `consulta`, limite a execucao a analise e resposta.
4. Se informou `bug`, `melhoria`, `evolucao` ou `nova funcionalidade`, encaminhe para a skill `plano-implementacao`.
5. Se o pedido atual contradiz a classificacao anterior, interrompa e solicite reclassificacao.

## Sinais de alerta

- Pedido classificado como `consulta` contendo verbos como "implementar", "alterar", "corrigir", "criar", "remover" ou "ajustar".
- Pedido classificado como `bug` sem expectativa de reproduzir o problema via teste.
- Pedido de implementacao sem plano aprovado pelo usuario.

## Resultado esperado

Ao final da triagem, o chat deve estar em um destes estados:

- demanda bloqueada aguardando classificacao;
- demanda bloqueada aguardando reclassificacao;
- demanda validada como `consulta`;
- demanda validada para seguir ao plano de implementacao.
