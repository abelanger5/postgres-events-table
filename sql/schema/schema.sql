-- CreateTable
CREATE TABLE simple_events (
    id BIGSERIAL NOT NULL,
    created_at TIMESTAMPTZ,
    tenant_id UUID NOT NULL,
    resource_id UUID NOT NULL,
    data JSONB,
    PRIMARY KEY (id)
);

-- Create index on resource_id and tenant_id
CREATE INDEX ON simple_events (tenant_id, resource_id);

-- CreateTable logs
CREATE TABLE logs (
    created_at TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    resource_id UUID NOT NULL,
    log TEXT
);

-- TODO: FIX THIS!
SELECT create_hypertable('logs', by_range('created_at'));

-- Create an index on the tenant and created_at columns
CREATE INDEX ON logs (tenant_id, resource_id, created_at);

-- Create enum for event type -- SUCCEEDED or FAILED
CREATE TYPE metric_event_type AS ENUM ('SUCCEEDED', 'FAILED');

-- CreateTable metric_events
CREATE TABLE metric_events (
    id BIGSERIAL NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    resource_id UUID NOT NULL,
    event_type metric_event_type NOT NULL,
    data JSONB,
    CONSTRAINT metric_events_pkey PRIMARY KEY (id, created_at)
);

-- Convert to hypertable
-- TODO: FIX THIS!
SELECT create_hypertable('metric_events', by_range('created_at'));

-- Create continuous aggregate view for metric_events
CREATE  MATERIALIZED VIEW metric_events_summary
   WITH (timescaledb.continuous)
   AS
      SELECT
        time_bucket('1 minute', created_at) AS minute,
        tenant_id,
        resource_id,
        COUNT(*) FILTER (WHERE event_type = 'SUCCEEDED') AS succeeded_count,
        COUNT(*) FILTER (WHERE event_type = 'FAILED') AS failed_count
      FROM metric_events
      GROUP BY minute, tenant_id, resource_id
      ORDER BY minute;

CREATE INDEX IF NOT EXISTS metric_events_summary__tenantId_resourceId_minute_idx ON metric_events_summary (tenant_id, resource_id, minute);

ALTER MATERIALIZED VIEW metric_events_summary set (timescaledb.materialized_only = true);

SELECT add_continuous_aggregate_policy('metric_events_summary',
  start_offset => NULL,
  end_offset => INTERVAL '15 minutes',
  schedule_interval => INTERVAL '15 minutes');
