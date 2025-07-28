// plugins/vector_plugin/pgvector_base.go
// Base structures and common functionality for comprehensive pgvector testing
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// ComprehensivePgVectorWorkload provides extensive pgvector testing capabilities
type ComprehensivePgVectorWorkload struct {
	TestType         string      // "ingestion", "update", "read", "index", "accuracy"
	IngestionMethod  string      // "single", "batch", "copy"
	BatchSize        int         // for batch operations
	ReadType         string      // "full_scan", "indexed"
	IndexType        string      // "ivfflat", "hnsw", "none"
	Dimensions       int         // vector dimensions (default 1024)
	SimilarityMetric string      // "l2", "cosine", "inner_product"
	PreloadedData    [][]float32 // 10% pre-calculated vectors for consistent testing
	BaselineRows     int         // number of rows to pre-load (default 1M)
}

// IndexConfiguration represents different index configurations to test
type IndexConfiguration struct {
	Name        string
	IndexType   string // "ivfflat", "hnsw"
	Parameters  map[string]interface{}
	Ops         string // "vector_l2_ops", "vector_cosine_ops", etc.
	Description string
}

// generateRandomVector creates a random vector
func (w *ComprehensivePgVectorWorkload) generateRandomVector(rng *rand.Rand) []float32 {
	vector := make([]float32, w.Dimensions)
	for i := range vector {
		vector[i] = rng.Float32()*2 - 1
	}
	return vector
}

// loadVectorsFromFile loads vectors from CSV file
func (w *ComprehensivePgVectorWorkload) loadVectorsFromFile(filename string) ([][]float32, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	vectors := make([][]float32, len(records))
	for i, record := range records {
		vector := make([]float32, len(record))
		for j, val := range record {
			if f, err := strconv.ParseFloat(val, 32); err == nil {
				vector[j] = float32(f)
			}
		}
		vectors[i] = vector
	}

	return vectors, nil
}

// saveVectorsToFile saves vectors to CSV file
func (w *ComprehensivePgVectorWorkload) saveVectorsToFile(filename string, vectors [][]float32) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, vector := range vectors {
		record := make([]string, len(vector))
		for j, val := range vector {
			record[j] = strconv.FormatFloat(float64(val), 'f', 6, 32)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// getIndexConfigurations returns all index configurations to test
func (w *ComprehensivePgVectorWorkload) getIndexConfigurations() []IndexConfiguration {
	var configs []IndexConfiguration

	// Determine operator class based on similarity metric
	var ops string
	switch w.SimilarityMetric {
	case "cosine":
		ops = "vector_cosine_ops"
	case "inner_product":
		ops = "vector_ip_ops"
	default:
		ops = "vector_l2_ops"
	}

	// IVFFlat configurations
	for _, lists := range []int{50, 100, 200, 500, 1000} {
		configs = append(configs, IndexConfiguration{
			Name:      fmt.Sprintf("ivfflat_lists_%d", lists),
			IndexType: "ivfflat",
			Parameters: map[string]interface{}{
				"lists": lists,
			},
			Ops:         ops,
			Description: fmt.Sprintf("IVFFlat with %d lists", lists),
		})
	}

	// HNSW configurations (if PostgreSQL version supports it)
	for _, m := range []int{8, 16, 32} {
		for _, efConstruction := range []int{64, 128, 256} {
			configs = append(configs, IndexConfiguration{
				Name:      fmt.Sprintf("hnsw_m_%d_ef_%d", m, efConstruction),
				IndexType: "hnsw",
				Parameters: map[string]interface{}{
					"m":               m,
					"ef_construction": efConstruction,
				},
				Ops:         ops,
				Description: fmt.Sprintf("HNSW with M=%d, EF_CONSTRUCTION=%d", m, efConstruction),
			})
		}
	}

	return configs
}

// parseWorkloadType extracts configuration from workload type string
func (w *ComprehensivePgVectorWorkload) parseWorkloadType(workloadType string) {
	// Default values
	w.Dimensions = 1024
	w.SimilarityMetric = "l2"
	w.BaselineRows = 100000 // Reduced for testing
	w.BatchSize = 500

	// Parse workload type: pgvector_{test_type}_{method}_{similarity}_{dimensions}
	// Example: pgvector_ingestion_batch_cosine_1024
	parts := strings.Split(workloadType, "_")

	if len(parts) >= 2 {
		w.TestType = parts[1] // ingestion, update, read, index, accuracy
	}

	if len(parts) >= 3 {
		switch parts[2] {
		case "single", "batch", "copy":
			w.IngestionMethod = parts[2]
		case "scan", "indexed":
			w.ReadType = "full_" + parts[2]
		case "ivfflat", "hnsw", "none":
			w.IndexType = parts[2]
		}
	}

	if len(parts) >= 4 {
		switch parts[3] {
		case "l2", "cosine", "inner":
			w.SimilarityMetric = parts[3]
			if parts[3] == "inner" {
				w.SimilarityMetric = "inner_product"
			}
		}
	}

	if len(parts) >= 5 {
		if dims, err := strconv.Atoi(parts[4]); err == nil {
			w.Dimensions = dims
		}
	}

	// Set defaults if not specified
	if w.TestType == "" {
		w.TestType = "ingestion"
	}
	if w.IngestionMethod == "" {
		w.IngestionMethod = "batch"
	}
	if w.ReadType == "" {
		w.ReadType = "indexed"
	}
	if w.IndexType == "" {
		w.IndexType = "ivfflat"
	}

	log.Printf("🔧 Parsed workload configuration:")
	log.Printf("   - Test Type: %s", w.TestType)
	log.Printf("   - Ingestion Method: %s", w.IngestionMethod)
	log.Printf("   - Read Type: %s", w.ReadType)
	log.Printf("   - Index Type: %s", w.IndexType)
	log.Printf("   - Similarity Metric: %s", w.SimilarityMetric)
	log.Printf("   - Dimensions: %d", w.Dimensions)
	log.Printf("   - Baseline Rows: %d", w.BaselineRows)
}
