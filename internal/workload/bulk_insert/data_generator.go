// internal/workload/bulk_insert/data_generator.go
package bulk_insert

import (
	"crypto/rand"
	"fmt"
	"math"
	mrand "math/rand"
	"net"
	"time"
)

// DataGenerator provides methods to generate realistic test data
// for bulk insert operations with various data patterns
type DataGenerator struct {
	rng *mrand.Rand
}

// NewDataGenerator creates a new data generator with optional seed
func NewDataGenerator(seed int64) *DataGenerator {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &DataGenerator{
		rng: mrand.New(mrand.NewSource(seed)),
	}
}

// GenerateRecord creates a single test record with realistic data
func (dg *DataGenerator) GenerateRecord() DataRecord {
	return DataRecord{
		ShortText:    dg.generateShortText(),
		MediumText:   dg.generateMediumText(),
		LongText:     dg.generateLongText(),
		IntValue:     dg.rng.Int31n(1000000),
		BigintValue:  dg.rng.Int63n(9223372036854775807),
		DecimalValue: dg.generateDecimal(),
		FloatValue:   dg.rng.Float64() * 1000000,
		EventDate:    dg.generateEventDate(),
		EventTime:    dg.generateEventTime(),
		IsActive:     dg.rng.Float32() < 0.8, // 80% active
		Metadata:     dg.generateMetadata(),
		DataBlob:     dg.generateBlob(),
		StatusEnum:   dg.generateStatusEnum(),
		Tags:         dg.generateTags(),
		ClientIP:     dg.generateIP(),
		LocationX:    dg.rng.Float64()*180 - 90,  // Latitude -90 to 90
		LocationY:    dg.rng.Float64()*360 - 180, // Longitude -180 to 180
	}
}

// GenerateBatch creates multiple records efficiently
func (dg *DataGenerator) GenerateBatch(count int) []DataRecord {
	records := make([]DataRecord, count)
	for i := 0; i < count; i++ {
		records[i] = dg.GenerateRecord()
	}
	return records
}

// generateShortText creates short text strings (10-50 characters)
func (dg *DataGenerator) generateShortText() string {
	prefixes := []string{"User", "Order", "Product", "Service", "Event", "Task", "Item", "Record"}
	prefix := prefixes[dg.rng.Intn(len(prefixes))]
	suffix := dg.rng.Intn(999999)
	return fmt.Sprintf("%s_%06d", prefix, suffix)
}

// generateMediumText creates medium text strings (50-500 characters)
func (dg *DataGenerator) generateMediumText() string {
	phrases := []string{
		"This is a sample text for testing bulk insert performance",
		"PostgreSQL is a powerful open-source relational database system",
		"Bulk insert operations are critical for data warehouse workloads",
		"Performance testing requires realistic data patterns and distributions",
		"Database indexing strategies significantly impact query performance",
		"ACID compliance ensures data integrity in concurrent environments",
		"Query optimization is essential for large-scale applications",
		"Monitoring and alerting help maintain system reliability",
	}
	
	// Combine 2-4 phrases with some variation
	numPhrases := 2 + dg.rng.Intn(3)
	result := ""
	for i := 0; i < numPhrases; i++ {
		if i > 0 {
			result += ". "
		}
		result += phrases[dg.rng.Intn(len(phrases))]
	}
	return result + "."
}

// generateLongText creates long text strings (500+ characters)
func (dg *DataGenerator) generateLongText() string {
	templates := []string{
		"In the realm of database performance testing, bulk insert operations represent one of the most challenging aspects of system optimization. The ability to efficiently insert large volumes of data directly impacts the overall throughput and scalability of database applications. Modern PostgreSQL installations must handle various data types, indexing strategies, and concurrent operations while maintaining ACID compliance and data integrity. Performance characteristics can vary significantly based on table structure, index configuration, storage parameters, and system resources. Understanding these factors is crucial for designing efficient data loading strategies.",
		"Enterprise data warehouse systems frequently encounter scenarios requiring the insertion of millions or billions of records within specific time windows. These operations must be optimized not only for raw throughput but also for minimal impact on concurrent read operations and system stability. Factors such as write-ahead logging, checkpoint frequency, vacuum operations, and memory buffer management all play critical roles in determining the overall performance profile. Additionally, the choice between different insertion methods - including individual INSERTs, batch INSERTs, and COPY operations - can dramatically affect performance outcomes.",
		"The evolution of storage technology, from traditional spinning disks to modern NVMe SSDs, has fundamentally changed the performance characteristics of bulk data operations. While sequential write performance has improved dramatically, the relationship between different PostgreSQL configuration parameters and optimal performance has also evolved. Modern systems must balance factors such as shared_buffers, work_mem, maintenance_work_mem, and wal_buffers to achieve optimal bulk insert performance. Understanding these relationships is essential for database administrators and application developers working with large-scale data operations.",
	}
	
	return templates[dg.rng.Intn(len(templates))]
}

// generateDecimal creates decimal values with realistic precision
func (dg *DataGenerator) generateDecimal() float64 {
	// Generate values with up to 4 decimal places
	base := dg.rng.Float64() * 100000
	return math.Round(base*10000) / 10000
}

// generateEventDate creates dates within the last year
func (dg *DataGenerator) generateEventDate() time.Time {
	now := time.Now()
	daysBack := dg.rng.Intn(365)
	return now.AddDate(0, 0, -daysBack)
}

// generateEventTime creates random times within a day
func (dg *DataGenerator) generateEventTime() time.Time {
	hour := dg.rng.Intn(24)
	minute := dg.rng.Intn(60)
	second := dg.rng.Intn(60)
	return time.Date(2000, 1, 1, hour, minute, second, 0, time.UTC)
}

// generateMetadata creates realistic JSON metadata
func (dg *DataGenerator) generateMetadata() map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Add some common metadata fields
	metadata["version"] = fmt.Sprintf("1.%d.%d", dg.rng.Intn(10), dg.rng.Intn(100))
	metadata["priority"] = []string{"low", "medium", "high", "critical"}[dg.rng.Intn(4)]
	metadata["source"] = []string{"web", "mobile", "api", "batch"}[dg.rng.Intn(4)]
	
	// Add some numeric data
	metadata["score"] = dg.rng.Float64() * 100
	metadata["attempts"] = dg.rng.Intn(10)
	
	// Occasionally add nested objects
	if dg.rng.Float32() < 0.3 {
		nested := make(map[string]interface{})
		nested["timestamp"] = time.Now().Unix()
		nested["user_agent"] = "Mozilla/5.0 (compatible; TestBot/1.0)"
		metadata["details"] = nested
	}
	
	return metadata
}

// generateBlob creates binary data of varying sizes
func (dg *DataGenerator) generateBlob() []byte {
	// Generate blobs of 0-1024 bytes
	size := dg.rng.Intn(1025)
	if size == 0 {
		return nil
	}
	
	blob := make([]byte, size)
	rand.Read(blob) // Use crypto/rand for better randomness
	return blob
}

// generateStatusEnum returns a random status value
func (dg *DataGenerator) generateStatusEnum() string {
	statuses := []string{"pending", "processing", "completed", "failed", "cancelled"}
	weights := []int{30, 20, 40, 8, 2} // Weighted distribution
	
	total := 0
	for _, w := range weights {
		total += w
	}
	
	r := dg.rng.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return statuses[i]
		}
	}
	return statuses[0] // Fallback
}

// generateTags creates an array of tag strings
func (dg *DataGenerator) generateTags() []string {
	allTags := []string{
		"urgent", "important", "batch", "realtime", "test", "production",
		"analytics", "reporting", "etl", "monitoring", "backup", "archive",
		"customer", "internal", "external", "automated", "manual", "scheduled",
	}
	
	// Generate 0-5 tags
	numTags := dg.rng.Intn(6)
	if numTags == 0 {
		return nil
	}
	
	tags := make([]string, numTags)
	used := make(map[int]bool)
	
	for i := 0; i < numTags; i++ {
		for {
			idx := dg.rng.Intn(len(allTags))
			if !used[idx] {
				tags[i] = allTags[idx]
				used[idx] = true
				break
			}
		}
	}
	
	return tags
}

// generateIP creates realistic IP addresses
func (dg *DataGenerator) generateIP() string {
	// Generate mostly IPv4 with some IPv6
	if dg.rng.Float32() < 0.9 {
		// IPv4
		return net.IPv4(
			byte(dg.rng.Intn(256)),
			byte(dg.rng.Intn(256)),
			byte(dg.rng.Intn(256)),
			byte(dg.rng.Intn(256)),
		).String()
	} else {
		// IPv6 (simplified)
		return fmt.Sprintf("2001:db8::%04x:%04x",
			dg.rng.Intn(65536),
			dg.rng.Intn(65536))
	}
}
