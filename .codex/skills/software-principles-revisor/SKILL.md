---
name: software-principles-revisor
description: Revisa código e aponta violações de princípios de software (SOLID, OO, Pragmáticos) na solução implementada
---

# 🔍 Revisor de Princípios de Software

Skill para análise de código e identificação de violações a princípios fundamentais de design de software.

**Referência:** Skill [`software-principles`](../software-principles/SKILL.md)

> [!IMPORTANT]
> **Filosofia Core:** Princípios são guias, não regras absolutas. Reporte violações com contexto e pragmatismo — nem toda violação precisa ser corrigida.

---

## 📋 Quando Usar

Execute esta skill quando:
- Após implementação pelo agente, antes de entregar ao usuário
- Antes de code review para identificar violações de princípios
- Após refatorações para validar que princípios foram mantidos

---

## 🎯 Instruções de Execução

### 1. Coletar Informações

Pergunte ao usuário:
1. **Diretório raiz do projeto** a ser analisado
2. **Linguagem principal** (Go, Java, Python, etc.)

Colete também o contexto minimo da entrega:
3. **Tipo da demanda** (`bug`, `melhoria`, `evolucao` ou `nova funcionalidade`)
4. **Arquivos alterados**
5. **Testes executados e resultado**
6. **Arquivos de teste criados, alterados ou removidos**

Se a entrega nao trouxer evidencia minima de testes, trate isso como risco relevante de qualidade.

### 2. Análise de Princípios SOLID

#### 2.1 Single Responsibility Principle (SRP)

> Uma classe deve ter apenas um motivo para mudar.

**Verificar:**
- ✅ Classe tem nome claro e específico
- ✅ Poucas dependências injetadas
- ✅ Testes focados e simples
- ✅ Tamanho razoável (~200 linhas)
- ❌ Nome genérico (`Manager`, `Helper`, `Utils`)
- ❌ Muitas dependências injetadas (5+)
- ❌ Testes precisam de muitos mocks
- ❌ Arquivo gigante (500+ linhas)
- ❌ Descrição do que a classe faz usa a palavra "E" (sinal de múltiplas responsabilidades)

#### 2.2 Open/Closed Principle (OCP)

> Aberto para extensão, fechado para modificação.

**Verificar:**
- ✅ Novos comportamentos são adicionados via extensão (Strategy, Decorator, polimorfismo)
- ✅ Código existente não precisa ser alterado para adicionar novo tipo
- ❌ `switch/if` que cresce a cada novo tipo/comportamento
- ❌ Modificação de código existente para cada variação nova
- ❌ Over-engineering preventivo (polimorfismo para 1-2 variações estáveis)

#### 2.3 Liskov Substitution Principle (LSP)

> Subtipos devem ser substituíveis por seus tipos base sem quebrar o programa.

**Verificar:**
- ✅ Subclasses honram o contrato do tipo base
- ✅ Pré-condições iguais ou mais fracas, pós-condições iguais ou mais fortes
- ❌ Subclasse lança exceção não esperada pelo contrato
- ❌ Subclasse ignora/sobrescreve comportamento com `no-op`
- ❌ Pré-condições mais restritivas ou pós-condições mais fracas
- ❌ Padrões como `Square extends Rectangle`

**Teste mental:** *"Posso usar a subclasse em QUALQUER lugar que o pai é usado sem surpresas?"*

#### 2.4 Interface Segregation Principle (ISP)

> Clientes não devem depender de interfaces que não utilizam.

**Verificar:**
- ✅ Interfaces com 3-5 métodos focados
- ✅ Todo implementador usa TODOS os métodos da interface
- ✅ Nome da interface descreve um papel/capacidade
- ❌ Interface com 10+ métodos
- ❌ Implementador com métodos `throw UnsupportedOperation` / `no-op`
- ❌ Nome genérico (`IService`, `IManager`)

#### 2.5 Dependency Inversion Principle (DIP)

> Módulos de alto nível não devem depender de módulos de baixo nível. Ambos devem depender de abstrações.

**Verificar:**
- ✅ Use Case / Application dependem de interfaces (ports)
- ✅ Domain não depende de nada externo
- ✅ Infrastructure implementa interfaces definidas pelo domínio
- ❌ Camada de alto nível importa classe concreta de infra
- ❌ Regras de negócio acopladas a tecnologia específica (banco, framework)

**Teste:** *"Se eu trocar o banco de dados, preciso alterar regras de negócio?"* → Se sim, DIP violado.

### 3. Análise de Princípios OO

#### 3.1 Lei de Demeter (LoD)

> Fale só com amigos imediatos, nunca com estranhos.

**Verificar:**
- ✅ Métodos chamam apenas: `this`, parâmetros, objetos criados internamente, atributos diretos
- ❌ Train wrecks: `a.getB().getC().doSomething()`
- ❌ Navegação profunda em grafos de objetos

#### 3.2 Tell, Don't Ask

> Diga aos objetos o que fazer; não pergunte seu estado para decidir por eles.

**Verificar:**
- ✅ Objetos encapsulam decisões sobre seu próprio estado
- ✅ Comportamento vive no objeto que detém os dados
- ❌ Getters usados para tomar decisões externas ao objeto
- ❌ Lógica de negócio fora do objeto que possui os dados (`if (obj.getX()) { ... }`)

#### 3.3 Composition over Inheritance

> Prefira composição à herança para reutilizar comportamento.

**Verificar:**
- ✅ Herança usada para relações `é-um` verdadeiras e estáveis
- ✅ Composição usada para relações `tem-um` ou `usa-um`
- ❌ Herança profunda (3+ níveis)
- ❌ Herança para reutilizar código (deveria ser composição/delegação)
- ❌ Fragilidade da classe base (mudança no pai quebra filhos)

#### 3.4 Program to an Interface

> Declare tipos usando interfaces/abstrações, não classes concretas.

**Verificar:**
- ✅ Variáveis e parâmetros declarados como interfaces
- ✅ Construtores recebem abstrações
- ❌ Dependências declaradas como classes concretas
- ❌ `new ConcreteClass()` espalhado pelo código de negócio

#### 3.5 Encapsulate What Varies

> Isole o que muda do que permanece estável.

**Verificar:**
- ✅ Pontos de variação isolados atrás de abstrações
- ✅ Mudanças frequentes encapsuladas em componentes separados
- ❌ Código que muda junto espalhado por múltiplas classes
- ❌ Pontos de variação expostos diretamente

#### 3.6 Principle of Least Astonishment (POLA)

> O código deve se comportar como o leitor espera.

**Verificar:**
- ✅ Nomes de métodos/classes refletem comportamento real
- ✅ Retornos e efeitos colaterais são previsíveis
- ❌ Método com efeito colateral inesperado (ex: getter que modifica estado)
- ❌ Nome que sugere uma coisa mas faz outra

### 4. Análise de Princípios Pragmáticos

#### 4.1 DRY — Don't Repeat Yourself

> Cada conhecimento, uma representação.

**Verificar:**
- ✅ Lógica de negócio centralizada em um único lugar
- ✅ Abstrações criadas após 3+ repetições (Rule of 3)
- ❌ Código duplicado com mesma lógica em múltiplos locais
- ❌ Abstração prematura (DRY antes de ter padrão claro)

#### 4.2 KISS — Keep It Simple

> A solução mais simples que funciona.

**Verificar:**
- ✅ Solução direta e fácil de entender
- ✅ Complexidade justificada por requisitos reais
- ❌ Abstrações desnecessárias (pattern para um único caso)
- ❌ Indirections que dificultam rastreamento do fluxo

#### 4.3 YAGNI — You Aren't Gonna Need It

> Não construa para requisitos especulativos.

**Verificar:**
- ✅ Código atende requisitos atuais
- ✅ Extensibilidade adicionada sob demanda
- ❌ Interfaces/abstrações sem segundo implementador
- ❌ Configurações, flags ou parâmetros para cenários inexistentes

#### 4.4 Boy Scout Rule

> Deixe o código mais limpo do que encontrou.

**Verificar:**
- ✅ Melhorias pontuais no código tocado
- ❌ Código tocado sem melhoria de clareza (nomes, estrutura)
- ❌ Refatoração massiva fora do escopo (escopo controlado)

#### 4.5 Fail Fast

> Detecte e reporte erros o mais cedo possível.

**Verificar:**
- ✅ Validações na entrada (parâmetros, DTOs)
- ✅ Erros tratados próximos à origem
- ❌ Erros silenciados (catch vazio, log sem ação)
- ❌ Validação tardia (no meio do fluxo ao invés da entrada)

---

## 📝 Formato do Relatório

Gere um relatório Markdown com:

```markdown
# Relatório de Revisão de Princípios de Software

**Projeto:** [nome/caminho]
**Data:** [data]
**Linguagem:** [linguagem]

## Resumo

| Categoria | Status | Problemas |
|-----------|--------|-----------|
| SRP — Single Responsibility | ✅/⚠️/❌ | X |
| OCP — Open/Closed | ✅/⚠️/❌ | X |
| LSP — Liskov Substitution | ✅/⚠️/❌ | X |
| ISP — Interface Segregation | ✅/⚠️/❌ | X |
| DIP — Dependency Inversion | ✅/⚠️/❌ | X |
| Lei de Demeter | ✅/⚠️/❌ | X |
| Tell, Don't Ask | ✅/⚠️/❌ | X |
| Composition over Inheritance | ✅/⚠️/❌ | X |
| Program to Interface | ✅/⚠️/❌ | X |
| Encapsulate What Varies | ✅/⚠️/❌ | X |
| POLA | ✅/⚠️/❌ | X |
| DRY | ✅/⚠️/❌ | X |
| KISS | ✅/⚠️/❌ | X |
| YAGNI | ✅/⚠️/❌ | X |
| Boy Scout Rule | ✅/⚠️/❌ | X |
| Fail Fast | ✅/⚠️/❌ | X |

## Problemas Encontrados

### [Categoria — Nome do Princípio]

#### [Problema 1]
- **Arquivo:** `caminho/arquivo.ext`
- **Linha:** X
- **Princípio:** [nome do princípio violado]
- **Descrição:** [descrição do problema]
- **Impacto:** [acoplamento, fragilidade, dificuldade de teste, etc.]
- **Sugestão:** [como corrigir, com exemplo de código quando aplicável]

## Recomendações Gerais

[Lista de melhorias sugeridas]

## Verificações de Fluxo

- **Tipo da demanda:** [tipo]
- **Testes executados:** [sim/não + evidência]
- **Alterações em testes:** [arquivos]
- **Risco de processo:** [nenhum / atenção / crítico]
```

---

## ⚠️ Observações

- **Contexto importa:** Nem toda violação técnica é um problema real. Avalie o impacto no contexto do projeto.
- **Pragmatismo:** Não reporte violações triviais que não impactam manutenibilidade (ex: YAGNI em código com 10 linhas).
- **Priorize:** Foque em violações que causam acoplamento, fragilidade ou dificuldade de teste.
- **Balanceamento:** SOLID e KISS/YAGNI podem conflitar — prefira simplicidade quando a complexidade não se justifica.
- Se a solução ficou "verde" as custas de enfraquecimento ou remocao indevida de testes, registre isso como problema grave de qualidade.

---

## 📚 Referências

- [Clean Code — Robert C. Martin](https://www.amazon.com/Clean-Code-Handbook-Software-Craftsmanship/dp/0132350882)
- [Agile Software Development — Robert C. Martin](https://www.amazon.com/Agile-Software-Development-Principles-Practices/dp/0135974445)
- [The Pragmatic Programmer — Hunt & Thomas](https://pragprog.com/titles/tpp20/the-pragmatic-programmer-20th-anniversary-edition/)
- [Object-Oriented Software Construction — Bertrand Meyer](https://www.amazon.com/Object-Oriented-Software-Construction-Bertrand-Meyer/dp/0136291554)

> 💡 **Lembre-se:** Princípios são bússola, não GPS. Reporte violações com pragmatismo e sempre considere o contexto.
