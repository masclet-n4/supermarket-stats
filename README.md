# supermarket-stats

Servicio Go que calcula y actualiza `product_stats` a partir del histórico de
precios de cada producto.

## Qué hace

- Lee supermercados habilitados, productos y precios desde PocketBase.
- Calcula medias, diferencias, desviación, mínimos, máximos y cambios de precio
  para ventanas de 7, 30, 90 y 356 días, además del histórico completo.
- Crea o actualiza un registro `product_stats` por producto.
- Guarda un job por ejecución con tipo `stats:<nombre-del-supermarket>`.
- Registra métricas de duración, throughput, productos procesados, precios
  cargados y logs de operaciones lentas.

## Cómo funciona

El scheduler revisa cada minuto el campo `supermarkets.stats_schedule`, usando
cron de cinco campos. Cuando coincide, inicia un job por supermercado y procesa
sus productos con un pool de workers (`STATS_WORKERS`).

Cada producto carga su histórico, calcula las estadísticas y crea o actualiza
su resultado en PocketBase. Los errores de un producto no detienen los demás.

## Configuración

```bash
cp .env.example .env
```

Edita `.env`:

```dotenv
POCKETBASE_URL=http://localhost:8090
POCKETBASE_IDENTITY=admin@example.com
POCKETBASE_PASSWORD=change-me
STATS_WORKERS=8
```

`STATS_WORKERS` controla la concurrencia por supermercado. Empieza con 8–16 y
ajústalo midiendo la carga de PocketBase.

## Ejecución

Servicio continuo, ejecutado según cron:

```bash
make run
```

Ejecutar un supermercado una vez:

```bash
make run-supermarket SUPERMARKET_ID=abc123
```

Ejecutar todos los supermercados habilitados una vez:

```bash
make run-all
```

Otros comandos:

```bash
make test
make build
make check
```

## Docker

Construir y ejecutar:

```bash
docker build -t supermarket-stats .
docker run --rm --env-file .env supermarket-stats
```

Para una ejecución manual:

```bash
docker run --rm --env-file .env supermarket-stats --run-all
docker run --rm --env-file .env supermarket-stats --run-supermarket abc123
```

## Cron

`stats_schedule` usa cinco campos:

```text
minuto hora día-del-mes mes día-de-la-semana
```

Ejemplo: `0 3 * * *` ejecuta a las 03:00 según la zona horaria del proceso.

## Jobs y métricas

Cada job se crea con estado `running` y termina como `completed`,
`completed_with_errors` o `failed`. El campo `details` incluye duración total,
productos procesados/fallidos, throughput, precios cargados y tiempos medios y
y máximos por producto.

Los logs muestran el arranque, autenticación, próxima ejecución de cada
supermercado, progreso y productos o peticiones de PocketBase lentos.

## Configuración

```bash
export POCKETBASE_URL=http://localhost:8090
export POCKETBASE_IDENTITY=admin@example.com
export POCKETBASE_PASSWORD='secret'
export STATS_WORKERS=8
```

`stats_schedule` usa cron de cinco campos, con precisión de minutos:

```text
minuto hora día-del-mes mes día-de-la-semana
```

Ejemplo: `0 3 * * *` ejecuta a las 03:00 según la zona horaria del proceso.

## Ejecución

```bash
go run ./cmd/stats-service
```

## Diseño

- Un proceso de estadísticas por supermercado y coincidencia de cron.
- Un job `product_stats` por proceso en `jobs`.
- Un pool de workers por productos.
- Errores de producto acumulados en memoria y guardados al finalizar.
- Estados finales: `completed`, `completed_with_errors` o `failed`.
- El cálculo usa `prices.bulk_price` y `prices.date`.
