# API de Benchmark (MVP)

Esta API executa benchmarks on-demand para medir saúde e performance do host do ETL, origem e destino.

## Executar benchmark

`POST /api/projects/:id/benchmarks/run`

### Body (opcional)
```json
{
  "probeIterations": 5,
  "enableWriteProbe": false,
  "includeHost": true,
  "includeOrigin": true,
  "includeDestination": true
}
```

### Regras
- Se nenhum `include*` for enviado, todos serão ativados.
- `probeIterations` default = 5, máximo = 100.
- `enableWriteProbe` default = `true`.
- Use `enableWriteProbe=false` quando não houver permissão de escrita no DB.

### Resposta (200)
#### Tipos do retorno
- `run_id`: `string` (UUID)
- `project_id`: `string`
- `status`: `string` (`ok`, `partial`, `error`)
- `error`: `string` (opcional)
- `started_at`: `string` (RFC3339)
- `ended_at`: `string` (RFC3339)
- `options`: `object`
- `metrics`: `object`
- `scores`: `object`

#### Estrutura de `options`
- `probe_iterations`: `number` (int)
- `enable_write_probe`: `boolean`
- `include_host`: `boolean`
- `include_origin`: `boolean`
- `include_destination`: `boolean`

#### Estrutura de `metrics`
- `host_etl`: `object` (opcional)
- `origin`: `object` (opcional)
- `destination`: `object` (opcional)

#### Estrutura de `host_etl`
- `cpu_cores`: `number` (int)
- `cpu_usage_pct`: `number` (float)
- `mem_total_bytes`: `number` (uint64)
- `mem_used_bytes`: `number` (uint64)
- `swap_total_bytes`: `number` (uint64, opcional)
- `swap_used_bytes`: `number` (uint64, opcional)
- `disk_total_bytes`: `number` (uint64, opcional)
- `disk_free_bytes`: `number` (uint64, opcional)

#### Estrutura de `origin` / `destination`
- `db_type`: `string`
- `db_version`: `string` (opcional)
- `conn_latency_ms`: `number` (float, opcional)
- `ping_latency_ms`: `number` (float, opcional)
- `probe_iterations`: `number` (int, opcional)
- `probe_qps`: `number` (float, opcional)
- `write_enabled`: `boolean`
- `write_latency_ms`: `number` (float, opcional)
- `errors`: `array[string]` (opcional)

#### Estrutura de `scores`
- `host_etl`: `number` (float, 0–100, opcional)
- `origin`: `number` (float, 0–100, opcional)
- `destination`: `number` (float, 0–100, opcional)

```json
{
  "run_id": "uuid",
  "project_id": "project-id",
  "status": "ok",
  "error": "",
  "started_at": "2026-02-08T12:00:00Z",
  "ended_at": "2026-02-08T12:00:03Z",
  "options": {
    "probe_iterations": 5,
    "enable_write_probe": false,
    "include_host": true,
    "include_origin": true,
    "include_destination": true
  },
  "metrics": {
    "host_etl": {
      "cpu_cores": 8,
      "cpu_usage_pct": 32.5,
      "mem_total_bytes": 17179869184,
      "mem_used_bytes": 8589934592,
      "swap_total_bytes": 2147483648,
      "swap_used_bytes": 0,
      "disk_total_bytes": 512000000000,
      "disk_free_bytes": 256000000000
    },
    "origin": {
      "db_type": "postgres",
      "db_version": "PostgreSQL 15.4",
      "conn_latency_ms": 25,
      "ping_latency_ms": 8,
      "probe_iterations": 5,
      "probe_qps": 70,
      "write_enabled": false,
      "errors": []
    },
    "destination": {
      "db_type": "postgres",
      "db_version": "PostgreSQL 15.4",
      "conn_latency_ms": 30,
      "ping_latency_ms": 10,
      "probe_iterations": 5,
      "probe_qps": 55,
      "write_enabled": false,
      "errors": []
    }
  },
  "scores": {
    "host_etl": 87.5,
    "origin": 82.0,
    "destination": 79.5
  }
}
```

### Status possíveis
- `ok`: todos os alvos coletados com sucesso
- `partial`: algum alvo falhou
- `error`: todos falharam ou nenhum alvo selecionado

---

## Listar benchmarks

`GET /api/projects/:id/benchmarks?limit=10`

### Resposta (200)
#### Tipos do retorno
- `array` de objetos:
  - `run_id`: `string` (UUID)
  - `status`: `string` (`ok`, `partial`, `error`)
  - `started_at`: `string` (RFC3339)
  - `ended_at`: `string` (RFC3339)
  - `scores`: `object` (mesma estrutura de `scores` acima)

```json
[
  {
    "run_id": "uuid",
    "status": "ok",
    "started_at": "2026-02-08T12:00:00Z",
    "ended_at": "2026-02-08T12:00:03Z",
    "scores": {
      "host_etl": 87.5,
      "origin": 82.0,
      "destination": 79.5
    }
  }
]
```

---

## Obter benchmark por ID

`GET /api/projects/:id/benchmarks/:runId`

### Resposta (200)
- Retorna o mesmo payload completo de `POST /benchmarks/run` (com a mesma estrutura e tipos).

---

## Exportar benchmark em PDF

`GET /api/projects/:id/benchmarks/:runId/report`

### Resposta (200)
- `Content-Type`: `application/pdf`
- Corpo: PDF do benchmark (download).

---

## Exportar histórico de benchmarks em PDF

`GET /api/projects/:id/benchmarks/report?limit=50`

### Parâmetros
- `limit` (opcional): `number` (int). Se omitido, exporta todos.

### Resposta (200)
- `Content-Type`: `application/pdf`
- Corpo: PDF do histórico (download).

---

## Observações
- Sem acesso ao host do banco, métricas de CPU/RAM/disco dos DBs não são coletadas.
- Para MySQL e Postgres, o write probe usa tabela temporária na sessão.
- Histórico é salvo em `logs/benchmarks/<project_id>/benchmark_<run_id>.json`.
