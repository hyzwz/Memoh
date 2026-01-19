package memory

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

type QdrantStore struct {
	client     *qdrant.Client
	collection string
	dimension  int
}

type qdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

func NewQdrantStore(baseURL, apiKey, collection string, dimension int, timeout time.Duration) (*QdrantStore, error) {
	host, port, useTLS, err := parseQdrantEndpoint(baseURL)
	if err != nil {
		return nil, err
	}
	if collection == "" {
		collection = "memory"
	}
	if dimension <= 0 {
		dimension = 1536
	}

	cfg := &qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: useTLS,
	}
	client, err := qdrant.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	store := &QdrantStore{
		client:     client,
		collection: collection,
		dimension:  dimension,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutOrDefault(timeout))
	defer cancel()
	if err := store.ensureCollection(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *QdrantStore) Upsert(ctx context.Context, points []qdrantPoint) error {
	if len(points) == 0 {
		return nil
	}
	qPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, point := range points {
		payload, err := qdrant.TryValueMap(point.Payload)
		if err != nil {
			return err
		}
		qPoints = append(qPoints, &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(point.ID),
			Vectors: qdrant.NewVectorsDense(point.Vector),
			Payload: payload,
		})
	}
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Wait:           qdrant.PtrOf(true),
		Points:         qPoints,
	})
	return err
}

func (s *QdrantStore) Search(ctx context.Context, vector []float32, limit int, filters map[string]interface{}) ([]qdrantPoint, []float64, error) {
	if limit <= 0 {
		limit = 10
	}
	filter := buildQdrantFilter(filters)
	results, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collection,
		Query:          qdrant.NewQueryDense(vector),
		Limit:          qdrant.PtrOf(uint64(limit)),
		Filter:         filter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, nil, err
	}

	points := make([]qdrantPoint, 0, len(results))
	scores := make([]float64, 0, len(results))
	for _, scored := range results {
		points = append(points, qdrantPoint{
			ID:      pointIDToString(scored.GetId()),
			Payload: valueMapToInterface(scored.GetPayload()),
		})
		scores = append(scores, float64(scored.GetScore()))
	}
	return points, scores, nil
}

func (s *QdrantStore) Get(ctx context.Context, id string) (*qdrantPoint, error) {
	result, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.collection,
		Ids:            []*qdrant.PointId{qdrant.NewIDUUID(id)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	point := result[0]
	return &qdrantPoint{
		ID:      pointIDToString(point.GetId()),
		Payload: valueMapToInterface(point.GetPayload()),
	}, nil
}

func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collection,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrant.NewPointsSelectorIDs([]*qdrant.PointId{qdrant.NewIDUUID(id)}),
	})
	return err
}

func (s *QdrantStore) List(ctx context.Context, limit int, filters map[string]interface{}) ([]qdrantPoint, error) {
	if limit <= 0 {
		limit = 100
	}
	filter := buildQdrantFilter(filters)
	points, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.collection,
		Limit:          qdrant.PtrOf(uint32(limit)),
		Filter:         filter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	result := make([]qdrantPoint, 0, len(points))
	for _, point := range points {
		result = append(result, qdrantPoint{
			ID:      pointIDToString(point.GetId()),
			Payload: valueMapToInterface(point.GetPayload()),
		})
	}
	return result, nil
}

func (s *QdrantStore) DeleteAll(ctx context.Context, filters map[string]interface{}) error {
	filter := buildQdrantFilter(filters)
	if filter == nil {
		return fmt.Errorf("delete all requires filters")
	}
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collection,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrant.NewPointsSelectorFilter(filter),
	})
	return err
}

func (s *QdrantStore) ensureCollection(ctx context.Context) error {
	exists, err := s.client.CollectionExists(ctx, s.collection)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: s.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(s.dimension),
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

func parseQdrantEndpoint(endpoint string) (string, int, bool, error) {
	if endpoint == "" {
		return "127.0.0.1", 6334, false, nil
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", 0, false, err
	}
	host := parsed.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}
	port := 6334
	if parsed.Port() != "" {
		parsedPort, err := strconv.Atoi(parsed.Port())
		if err != nil {
			return "", 0, false, err
		}
		port = parsedPort
	}
	useTLS := parsed.Scheme == "https"
	return host, port, useTLS, nil
}

func timeoutOrDefault(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 10 * time.Second
	}
	return timeout
}

func buildQdrantFilter(filters map[string]interface{}) *qdrant.Filter {
	if len(filters) == 0 {
		return nil
	}
	conditions := make([]*qdrant.Condition, 0, len(filters))
	for key, value := range filters {
		if condition := buildQdrantCondition(key, value); condition != nil {
			conditions = append(conditions, condition)
		}
	}
	if len(conditions) == 0 {
		return nil
	}
	return &qdrant.Filter{
		Must: conditions,
	}
}

func buildQdrantCondition(key string, value interface{}) *qdrant.Condition {
	switch typed := value.(type) {
	case string:
		return qdrant.NewMatch(key, typed)
	case bool:
		return qdrant.NewMatchBool(key, typed)
	case int:
		return qdrant.NewMatchInt(key, int64(typed))
	case int64:
		return qdrant.NewMatchInt(key, typed)
	case float32:
		v := float64(typed)
		return qdrant.NewRange(key, &qdrant.Range{Gte: &v, Lte: &v})
	case float64:
		return qdrant.NewRange(key, &qdrant.Range{Gte: &typed, Lte: &typed})
	case map[string]interface{}:
		rangeValue := &qdrant.Range{}
		for _, op := range []string{"gte", "gt", "lte", "lt"} {
			if raw, ok := typed[op]; ok {
				val, ok := toFloat(raw)
				if !ok {
					continue
				}
				switch op {
				case "gte":
					rangeValue.Gte = &val
				case "gt":
					rangeValue.Gt = &val
				case "lte":
					rangeValue.Lte = &val
				case "lt":
					rangeValue.Lt = &val
				}
			}
		}
		if rangeValue.Gte != nil || rangeValue.Gt != nil || rangeValue.Lte != nil || rangeValue.Lt != nil {
			return qdrant.NewRange(key, rangeValue)
		}
	}
	return qdrant.NewMatch(key, fmt.Sprint(value))
}

func toFloat(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func pointIDToString(id *qdrant.PointId) string {
	if id == nil {
		return ""
	}
	if uuid := id.GetUuid(); uuid != "" {
		return uuid
	}
	if num := id.GetNum(); num != 0 {
		return fmt.Sprintf("%d", num)
	}
	return ""
}

func valueMapToInterface(values map[string]*qdrant.Value) map[string]interface{} {
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		result[key] = valueToInterface(value)
	}
	return result
}

func valueToInterface(value *qdrant.Value) interface{} {
	if value == nil {
		return nil
	}
	switch kind := value.GetKind().(type) {
	case *qdrant.Value_NullValue:
		return nil
	case *qdrant.Value_BoolValue:
		return kind.BoolValue
	case *qdrant.Value_IntegerValue:
		return kind.IntegerValue
	case *qdrant.Value_DoubleValue:
		return kind.DoubleValue
	case *qdrant.Value_StringValue:
		return kind.StringValue
	case *qdrant.Value_StructValue:
		return valueMapToInterface(kind.StructValue.GetFields())
	case *qdrant.Value_ListValue:
		items := make([]interface{}, 0, len(kind.ListValue.GetValues()))
		for _, item := range kind.ListValue.GetValues() {
			items = append(items, valueToInterface(item))
		}
		return items
	default:
		return nil
	}
}
