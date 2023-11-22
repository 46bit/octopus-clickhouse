CREATE TABLE IF NOT EXISTS octopus_30m_data (
  fuel Enum('gas' = 1, 'electricity' = 2),
  start DateTime,
  consumption Float64
) ENGINE = ReplacingMergeTree
PARTITION BY toYYYYMM(start)
ORDER BY (fuel, start)
PRIMARY KEY (fuel, start);

CREATE VIEW IF NOT EXISTS octopus_gas_30m AS (
  SELECT
    start,
    start + INTERVAL '30' MINUTE AS end,
    max(consumption) AS cubic_metres,
    cubic_metres * 10 AS approx_kwh
  FROM octopus_30m_data
  WHERE fuel = 'gas'
  GROUP BY start
  ORDER BY start
);

CREATE VIEW IF NOT EXISTS octopus_electricity_30m AS (
  SELECT
    start,
    start + INTERVAL '30' MINUTE AS end,
    max(consumption) AS kwh
  FROM octopus_30m_data
  WHERE fuel = 'electricity'
  GROUP BY start
  ORDER BY start
);
