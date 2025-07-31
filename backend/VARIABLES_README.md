# Sistema de Variáveis - ETL

## Visão Geral

O sistema de variáveis permite definir valores reutilizáveis a nível de projeto que podem ser utilizados nas queries SQL dos jobs. Isso facilita a manutenção e permite maior flexibilidade na execução dos pipelines.

## Como Funciona

### 1. Definição de Variáveis
As variáveis são definidas no projeto e armazenadas junto com os demais dados do projeto no arquivo `project.json`.

### 2. Sintaxe nas Queries
Nas queries SQL dos jobs, você pode referenciar as variáveis usando a sintaxe:
```sql
${nome_da_variavel}
```

### 3. Substituição durante Execução
Durante a execução dos jobs, o sistema automaticamente substitui os placeholders `${nome_da_variavel}` pelos valores reais definidos no projeto.

## Endpoints da API

### Listar Variáveis
```
GET /projects/{id}/variables
```

### Criar Variável
```
POST /projects/{id}/variables
Content-Type: application/json

{
  "name": "data_inicio",
  "value": "2024-01-01", 
  "description": "Data de início para filtros"
}
```

### Obter Variável Específica
```
GET /projects/{id}/variables/{variableName}
```

### Atualizar Variável
```
PUT /projects/{id}/variables/{variableName}
Content-Type: application/json

{
  "name": "data_inicio",
  "value": "2024-02-01",
  "description": "Data de início atualizada"
}
```

### Deletar Variável
```
DELETE /projects/{id}/variables/{variableName}
```

## Exemplo de Uso

### 1. Definir Variáveis no Projeto
```json
{
  "variables": [
    {
      "name": "data_inicio",
      "value": "2024-01-01",
      "description": "Data de início para filtros"
    },
    {
      "name": "tabela_origem", 
      "value": "vendas_2024",
      "description": "Nome da tabela de origem"
    },
    {
      "name": "limite_registros",
      "value": "1000",
      "description": "Limite de registros"
    }
  ]
}
```

### 2. Usar Variáveis nas Queries
```sql
-- Query de seleção usando variáveis
SELECT * FROM ${tabela_origem} 
WHERE data_venda >= '${data_inicio}' 
LIMIT ${limite_registros};

-- Query de inserção usando variáveis  
INSERT INTO destino_${tabela_origem} 
SELECT * FROM ${tabela_origem} 
WHERE data_venda >= '${data_inicio}';
```

### 3. Durante a Execução
O sistema substituirá automaticamente:
- `${tabela_origem}` → `vendas_2024`
- `${data_inicio}` → `2024-01-01`  
- `${limite_registros}` → `1000`

Resultando nas queries finais:
```sql
SELECT * FROM vendas_2024 
WHERE data_venda >= '2024-01-01' 
LIMIT 1000;

INSERT INTO destino_vendas_2024 
SELECT * FROM vendas_2024 
WHERE data_venda >= '2024-01-01';
```

## Tipos de Jobs Suportados

O sistema de variáveis funciona com todos os tipos de jobs:

- **Insert Jobs**: Substitui variáveis tanto no `selectSQL` quanto no `insertSQL`
- **Execution Jobs**: Substitui variáveis no `selectSQL`
- **Condition Jobs**: Substitui variáveis no `selectSQL`

## Vantagens

1. **Reutilização**: Defina uma vez, use em múltiplos jobs
2. **Manutenção**: Altere o valor em um local e afete todos os jobs
3. **Configuração**: Diferentes ambientes podem ter diferentes valores
4. **Flexibilidade**: Facilita parametrização de queries complexas
5. **Consistência**: Garante que todos os jobs usem os mesmos valores

## Considerações

- Os nomes das variáveis devem ser únicos dentro do projeto
- As variáveis são carregadas uma vez no início da execução do projeto
- Alterações nas variáveis só têm efeito na próxima execução do projeto
- Use nomes descritivos para as variáveis para facilitar a manutenção
