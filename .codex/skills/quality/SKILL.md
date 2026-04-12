---
name: quality-assurance
description: Regras para TDD, gate de testes antes e depois da implementacao, avaliacao de qualidade dos testes e proibicao de remover testes apenas para fazer a suite passar.
---

# 🧪 Regras de Testes e Qualidade

## 🎯 Cobertura Minima

- Todo novo artefato exige teste correspondente.
- `bug`, `melhoria`, `evolucao` e `nova funcionalidade` exigem testes unitarios.
- Em `bug`, o teste deve reproduzir o problema e impedir recorrencia.
- Foco principal: testes unitarios. Integracao entra em pontos criticos quando necessario.
- Testes de API: usar mocks para HTTP/DB quando o contexto pedir isolamento.

## 🚀 Fluxo Obrigatorio

### Antes de implementar

- Execute a suite de testes relevante.
- Se a suite falhar antes da mudanca, interrompa o trabalho.
- Analise a falha e discuta com o usuario antes de seguir.

### Durante a implementacao

- Nao trate teste como acessorio decorativo.
- Nao remova teste so para fazer a suite passar.
- Se um teste precisar ser removido por mudanca real de comportamento, registre a justificativa explicitamente.

### Ao final da implementacao

- Execute os testes relevantes e reporte `PASS` ou `FAIL`.
- Avalie a qualidade dos testes criados ou alterados.
- Verifique se os testes cobrem o comportamento pedido, nao apenas a linha feliz cosmetica.

## 🔍 Criterios de qualidade dos testes

- O teste falha sem a implementacao correta.
- O teste protege regra de negocio ou comportamento observavel.
- O teste tem nome claro e intencao legivel.
- O teste nao depende de excesso de mocks sem necessidade.
- O teste nao foi afrouxado artificialmente para aceitar comportamento incorreto.

## 🚫 Antipadroes proibidos

- Remover teste para maquiar suite verde.
- Alterar assercoes para algo vago so porque a implementacao nao sustentou o contrato.
- Declarar "nao aplicavel" para testes. Nesta base, teste sempre e aplicavel.

## ✅ Definition of Done (Geral)

1. Código limpo e testado.
2. Sem nomes genéricos.
3. Logs estruturados em inglês via `logAdapter`.
4. Testes executados antes e depois da implementacao.
5. Qualidade dos testes analisada e reportada.
6. Documentação sugerida/atualizada.
