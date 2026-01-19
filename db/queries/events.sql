-- name: InsertLifecycleEvent :exec
INSERT INTO lifecycle_events (id, container_id, event_type, payload)
VALUES (
  sqlc.arg(id),
  sqlc.arg(container_id),
  sqlc.arg(event_type),
  sqlc.arg(payload)
);

-- name: ListLifecycleEventsByContainerID :many
SELECT * FROM lifecycle_events WHERE container_id = sqlc.arg(container_id) ORDER BY created_at ASC;
