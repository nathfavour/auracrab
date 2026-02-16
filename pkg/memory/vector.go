package memory

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/nathfavour/auracrab/pkg/config"
)

type VectorEntry struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Embedding []float64              `json:"embedding"`
}

type VectorStore struct {
	entries []VectorEntry
	path    string
	mu      sync.RWMutex
}

func NewVectorStore(name string) (*VectorStore, error) {
	dir := filepath.Join(config.DataDir(), "memory")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, name+"_vectors.json")
	vs := &VectorStore{
		entries: []VectorEntry{},
		path:    path,
	}

	if err := vs.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return vs, nil
}

func (vs *VectorStore) Add(id, content string, metadata map[string]interface{}, embedding []float64) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.entries = append(vs.entries, VectorEntry{
		ID:        id,
		Content:   content,
		Metadata:  metadata,
		Embedding: embedding,
	})
	return vs.save()
}

func (vs *VectorStore) Search(queryEmbedding []float64, topK int) []VectorEntry {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	type result struct {
		entry VectorEntry
		score float64
	}

	results := []result{}
	for _, entry := range vs.entries {
		score := CosineSimilarity(queryEmbedding, entry.Embedding)
		results = append(results, result{entry, score})
	}

	// Sort by score descending (simple insertion sort for now)
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	final := []VectorEntry{}
	for i := 0; i < topK && i < len(results); i++ {
		final = append(final, results[i].entry)
	}

	return final
}

func (vs *VectorStore) load() error {
	f, err := os.ReadFile(vs.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(f, &vs.entries)
}

func (vs *VectorStore) save() error {
	data, err := json.MarshalIndent(vs.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vs.path, data, 0644)
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
