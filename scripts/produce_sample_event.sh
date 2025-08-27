#!/usr/bin/env bash
set -euo pipefail

BROKERS="localhost:9092"
TOPIC="pipeline.prepare_reviews.request"

# Sample event based on database row data:
MSG='{
  "message_id": "630394bf-625e-4f53-b37c-0f0f5ce2ef39",
  "saga_id": "36ed7aca-7804-4c4c-8e79-5485b69ff836",
  "type": "pipeline.prepare_reviews.request",
  "occurred_at": "2025-08-24T04:22:34.554363Z",
  "payload": {
    "app_id": "1074367771",
    "app_name": "welltory-heart-rate-monitor",
    "countries": ["us", "gb"],
    "date_from": "2025-08-17",
    "date_to": "2025-08-24"
  },
  "meta": {
    "app_id": "1074367771",
    "initiator": "system",
    "schema_version": "v1"
  }
}'

echo "Producing message to topic: $TOPIC"
echo "$MSG" | docker exec kafka rpk topic produce "$TOPIC" --brokers localhost:9092

echo "[✓] Sent sample event to $TOPIC"
