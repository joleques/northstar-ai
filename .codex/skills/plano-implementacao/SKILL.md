---
name: plano-implementacao
description: Gera o plano obrigatorio antes de implementar, detalhando o que vai mudar e onde vai mudar para bug e evolucao, e bloqueia a execucao ate aprovacao explicita do usuario.
---

# Plano de Implementacao

Use esta skill em toda demanda classificada como `bug`, `melhoria`, `evolucao` ou `nova funcionalidade`.

## Objetivo

Criar um plano claro, verificavel e aprovavel pelo usuario antes de qualquer implementacao.

## Regras obrigatorias

- Nenhuma implementacao comeca sem plano.
- Nenhuma implementacao comeca sem aprovacao explicita do usuario.
- O plano deve ser detalhado o suficiente para que o usuario consiga revisar a abordagem proposta.

## Estrutura do plano

### Para `bug`

- Descrever o problema a ser corrigido.
- Informar o que vai mudar e onde vai mudar.
- Informar quais testes unitarios vao reproduzir o problema e impedir recorrencia.
- Informar riscos de regressao ou pontos de impacto.

### Para `melhoria`

- Descrever o comportamento atual e a melhoria pretendida.
- Informar o que vai mudar.
- Informar testes unitarios que vao proteger o comportamento entregue.

### Para `evolucao`

- Descrever o objetivo da evolucao.
- Informar o que vai mudar e onde vai mudar.
- Informar impactos esperados em codigo, contratos ou fluxos.
- Informar testes unitarios que vao proteger o novo comportamento.

### Para `nova funcionalidade`

- Descrever o que sera entregue.
- Informar os componentes principais que serao criados ou adaptados, quando isso ja for conhecido.
- Informar testes unitarios que vao validar a funcionalidade.

## Gate de aprovacao

- O agente deve apresentar o plano e aguardar o usuario informar explicitamente que ele esta aprovado.
- Se o usuario pedir ajuste, o agente deve revisar o plano e reapresenta-lo.
- Enquanto nao houver aprovacao explicita, a execucao deve permanecer bloqueada.

## Gate de testes antes de implementar

Depois da aprovacao do plano e antes de editar arquivos:

1. rodar a suite de testes relevante;
2. se houver falha pre-existente, interromper a implementacao;
3. analisar a falha;
4. discutir com o usuario antes de seguir.

## Resultado esperado

Ao final desta skill, o chat deve ter:

- plano aprovado explicitamente pelo usuario;
- escopo de mudanca delimitado;
- testes previstos para a entrega;
- autorizacao para iniciar a implementacao.
