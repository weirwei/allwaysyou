package vector

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// DocumentMetadata represents structured metadata for a document
type DocumentMetadata struct {
	SessionID  string  `json:"session_id"`
	Role       string  `json:"role"`
	Category   string  `json:"category"`
	Source     string  `json:"source"`
	Importance float32 `json:"importance"`
	IsActive   bool    `json:"is_active"` // 是否有效(未被取代)
	CreatedAt  int64   `json:"created_at"`
}

// Document represents a document with its embedding
type Document struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Embedding []float32         `json:"embedding"`
	Metadata  map[string]string `json:"metadata"`            // 保持向后兼容
	MetaData  *DocumentMetadata `json:"meta_data,omitempty"` // 新的结构化元数据
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document Document `json:"document"`
	Score    float32  `json:"score"`
}

// SearchFilter represents advanced search filter options
type SearchFilter struct {
	SessionID  string   // 按会话过滤
	Categories []string // 按类别过滤
	ActiveOnly bool     // 仅返回有效记忆
	MinScore   float32  // 最低相似度
}

// VectorStore is an in-memory vector store with persistence
type VectorStore struct {
	documents map[string]Document
	mutex     sync.RWMutex
	path      string
}

// NewVectorStore creates a new vector store
func NewVectorStore(path string) (*VectorStore, error) {
	store := &VectorStore{
		documents: make(map[string]Document),
		path:      path,
	}

	// Ensure directory exists
	if path != "" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		// Load existing data if available
		if err := store.load(); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load data: %w", err)
		}
	}

	return store, nil
}

// Add adds a document to the store
func (s *VectorStore) Add(doc Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.documents[doc.ID] = doc

	// Persist to disk
	return s.save()
}

// AddBatch adds multiple documents to the store
func (s *VectorStore) AddBatch(docs []Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, doc := range docs {
		s.documents[doc.ID] = doc
	}

	return s.save()
}

// Get retrieves a document by ID
func (s *VectorStore) Get(id string) (Document, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	doc, ok := s.documents[id]
	return doc, ok
}

// Delete removes a document by ID
func (s *VectorStore) Delete(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.documents, id)
	return s.save()
}

// DeleteByMetadata removes documents matching metadata criteria
func (s *VectorStore) DeleteByMetadata(key, value string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for id, doc := range s.documents {
		if doc.Metadata[key] == value {
			delete(s.documents, id)
		}
	}

	return s.save()
}

// Search performs a similarity search using cosine similarity
func (s *VectorStore) Search(queryEmbedding []float32, limit int, filter map[string]string) []SearchResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []SearchResult

	for _, doc := range s.documents {
		// Apply filter
		if filter != nil {
			match := true
			for k, v := range filter {
				if doc.Metadata[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		score := cosineSimilarity(queryEmbedding, doc.Embedding)
		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// SearchWithFilter performs a similarity search with advanced filtering
func (s *VectorStore) SearchWithFilter(queryEmbedding []float32, limit int, filter *SearchFilter) []SearchResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []SearchResult

	for _, doc := range s.documents {
		// Apply structured filter
		if filter != nil {
			// Check active only (using new MetaData or fallback to old metadata)
			if filter.ActiveOnly {
				if doc.MetaData != nil && !doc.MetaData.IsActive {
					continue
				}
				// Fallback: check old metadata for is_active
				if doc.MetaData == nil && doc.Metadata["is_active"] == "false" {
					continue
				}
			}

			// Check session ID
			if filter.SessionID != "" {
				sessionID := ""
				if doc.MetaData != nil {
					sessionID = doc.MetaData.SessionID
				} else {
					sessionID = doc.Metadata["session_id"]
				}
				if sessionID != filter.SessionID {
					continue
				}
			}

			// Check categories
			if len(filter.Categories) > 0 {
				category := ""
				if doc.MetaData != nil {
					category = doc.MetaData.Category
				} else {
					category = doc.Metadata["category"]
				}
				found := false
				for _, c := range filter.Categories {
					if c == category {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
		}

		score := cosineSimilarity(queryEmbedding, doc.Embedding)

		// Check minimum score
		if filter != nil && filter.MinScore > 0 && score < filter.MinScore {
			continue
		}

		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// UpdateMetadata updates the metadata of a document
func (s *VectorStore) UpdateMetadata(id string, metadata *DocumentMetadata) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	doc, ok := s.documents[id]
	if !ok {
		return fmt.Errorf("document not found: %s", id)
	}

	doc.MetaData = metadata
	// Also update old metadata for backward compatibility
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]string)
	}
	doc.Metadata["is_active"] = fmt.Sprintf("%v", metadata.IsActive)
	doc.Metadata["category"] = metadata.Category
	doc.Metadata["source"] = metadata.Source

	s.documents[id] = doc
	return s.save()
}

// Count returns the number of documents
func (s *VectorStore) Count() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.documents)
}

// save persists the store to disk
func (s *VectorStore) save() error {
	if s.path == "" {
		return nil
	}

	data, err := json.Marshal(s.documents)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// load loads the store from disk
func (s *VectorStore) load() error {
	if s.path == "" {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &s.documents); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}
