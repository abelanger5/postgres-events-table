This repository accompanies [this blog post](https://docs.hatchet.run/blog/postgres-events-table).

## Setup

**Prerequisites:**

- [Taskfile](https://taskfile.dev/)
- Go 1.21+
- Docker Compose

Run `task setup` to get everything running. This will spin up a Postgres database on port 5432, generate the relevant `sqlc`, and write the schema to the database. You might need to run it multiple times if Postgres doesn't start quickly.

Next, set the `DATABASE_URL` environment variable for all commands below:

```
export DATABASE_URL=postgresql://hatchet:hatchet@127.0.0.1:5432/hatchet
```

Run the following command to compile the `pg-events` command:

```sh
go build -o ./bin/pg-events .
chmod +x ./bin/pg-events
export PATH=$PATH:$(pwd)/bin
```

### `pg-events simple`

Seeds the database with [simple events](./sql/schema/schema.sql#L2). For example, `pg-events simple -t 10 -r 100000 -c 10`. This would seed events across 10 tenants, 100k resource IDs, with 10 events per (tenant, resource) tuple.

```
simple inserts events into the simple table.

Usage:
  pg-events simple [flags]

Flags:
  -c, --count int       The number of events to create per (tenant, resource) tuple. (default 10)
  -h, --help            help for simple
  -r, --resources int   The number of distinct resources to create per tenant. (default 1000)
  -t, --tenants int     The number of tenants. (default 10)

Use "events simple [command] --help" for more information about a command.
```

### `pg-events simple query`

Runs a set of sampling queries across the simple events table to get timing information. For example, `pg-events simple query`.

```
query runs random selects on the simple_events table

Usage:
  pg-events simple query [flags]

Flags:
  -h, --help          help for query
  -s, --samples int   The number of random selects to run. (default 1000)

Global Flags:
  -c, --count int       The number of events to create per (tenant, resource) tuple. (default 10)
  -r, --resources int   The number of distinct resources to create per tenant. (default 1000)
  -t, --tenants int     The number of tenants. (default 10)
```

### `pg-events metrics`

Inserts random metrics data in the `metric_events` table. For example, `pg-events metrics -t 20 -r 20 -c 2000` will seed metrics across 20 tenants, 20 resources and 2000 metrics per (tenant, resource) tuple.

```
metrics inserts events for metrics into the metric_events table.

Usage:
  pg-events metrics [flags]

Flags:
  -c, --count int       The number of events to create per (tenant, resource) tuple. (default 1000)
  -h, --help            help for metrics
  -r, --resources int   The number of distinct resources to create per tenant. (default 10)
  -t, --tenants int     The number of tenants. (default 10)
```

### `pg-events logs`

Inserts random log data into the `logs` table. For example, `pg-events logs -t 10 -r 1 -c 10000` will seed logs across 10 tenants, 1 resource and 10k logs per resource.

```
logs inserts logs into the logs table.

Usage:
  pg-events logs [flags]

Flags:
  -c, --count int       The number of logs to create per (tenant, resource) tuple. (default 10000)
  -2, --end string      The end time for the logs. (default "2025-01-01T00:00:00Z")
  -h, --help            help for logs
  -r, --resources int   The number of distinct resources to create per tenant. (default 1)
  -1, --start string    The start time for the logs. (default "2024-01-01T00:00:00Z")
  -t, --tenants int     The number of tenants. (default 10)
```
