CREATE TABLE IF NOT EXISTS octopus_30m_data (
  fuel LowCardinality(String),
  start DateTime,
  inserted SimpleAggregateFunction(min, DateTime) DEFAULT now(),
  consumption SimpleAggregateFunction(max, Float64)
) ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(start)
ORDER BY (fuel, start)
PRIMARY KEY (fuel, start);

CREATE VIEW IF NOT EXISTS octopus_30m AS (
  SELECT
    fuel,
    start,
    start + INTERVAL '30' MINUTE AS end,
    min(inserted) AS inserted,
    max(consumption) AS consumption
  FROM octopus_30m_data
  GROUP BY fuel, start
  ORDER BY fuel, start
);

CREATE VIEW IF NOT EXISTS octopus_electricity_30m AS (
  SELECT
    start,
    end,
    inserted,
    consumption AS kwh
  FROM octopus_30m
  WHERE fuel = 'electricity'
);

CREATE VIEW IF NOT EXISTS octopus_gas_30m AS (
  SELECT
    start,
    end,
    inserted,
    consumption AS cubic_metres,
    cubic_metres * 10 AS approx_kwh
  FROM octopus_30m
  WHERE fuel = 'gas'
);
