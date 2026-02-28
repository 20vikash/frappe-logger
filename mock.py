import json
import random
import requests
from datetime import datetime, timedelta, timezone

QUICKWIT_URL = "http://10.0.0.2:7280/api/v1/logs/ingest"
# Change the host according to your quickwit server configuration.

DAYS = 5
TENANTS = ["1234", "ABCD", "EFGH"]
#Each tenant_id maps to a log_user DOCTYPE record.
LOGS_PER_RUN = 500

now = datetime.now(timezone.utc)
lines = []

for day_offset in range(DAYS):
    base_day = now - timedelta(days=day_offset)

    for tenant in TENANTS:
        for _ in range(LOGS_PER_RUN):
            timestamp = (
                base_day
                .replace(
                    hour=random.randint(0, 23),
                    minute=random.randint(0, 59),
                    second=random.randint(0, 59),
                    microsecond=0,
                )
                .isoformat()
            )

            log = {
                "time": timestamp,
                "tenant_id": tenant,
                "container_id": f"{tenant}_container_{random.randint(1,3)}",
                "container_name": f"{tenant}_service",
                "stream": random.choice(["stdout", "stderr"]),
                "log": f"Test log message {random.randint(1,100000)}",
            }

            lines.append(json.dumps(log))

ndjson_payload = "\n".join(lines)

print(f"Generated {len(lines)} logs. Sending to Quickwit...")

response = requests.post(
    QUICKWIT_URL,
    data=ndjson_payload,
    headers={"Content-Type": "application/x-ndjson"},
)

print("Status:", response.status_code)
print(response.text)

#NOTE
#For this to work, the schema of the quickwit index should be as follows:
#Name of the index: logs

# {
#   "version": "0.9",
#   "index_id": "logs",
#   "doc_mapping": {
#     "mode": "dynamic",
#     "field_mappings": [
#       {
#         "name": "time",
#         "type": "datetime",
#         "indexed": true,
#         "stored": true,
#         "fast": true,
#         "input_formats": ["rfc3339"],
#         "output_format": "rfc3339"
#       },
#       {
#         "name": "container_id",
#         "type": "text",
#         "indexed": true,
#         "stored": true,
#         "fast": {
#           "normalizer": "raw"
#         }
#       },
#       {
#         "name": "container_name",
#         "type": "text",
#         "indexed": true,
#         "stored": true,
#         "fast": {
#           "normalizer": "raw"
#         }
#       },
#       {
#         "name": "stream",
#         "type": "text",
#         "indexed": true,
#         "stored": true,
#         "fast": {
#           "normalizer": "raw"
#         }
#       },
#       {
#         "name": "log",
#         "type": "text",
#         "indexed": true,
#         "stored": true
#       }
#     ],
#     "timestamp_field": "time",
#     "partition_key": "container_id",
#     "max_num_partitions": 200
#   },
#   "indexing_settings": {
#     "commit_timeout_secs": 60
#   },
#   "ingest_settings": {
#     "min_shards": 1
#   }
# }
