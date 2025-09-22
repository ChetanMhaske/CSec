package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

// This struct is for data coming FROM the agent
type SecurityEvent struct {
	Timestamp string `json:"timestamp"`
	Hostname  string `json:"hostname"`
	EventType string `json:"event_type"`
	Details   string `json:"details"`
}

// This struct is for data being SENT TO the frontend, with JSON and DB tags.
type SecurityEventForFrontend struct {
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Hostname  string    `json:"hostname"  db:"hostname"`
	EventType string    `json:"event_type" db:"event_type"`
	Details   string    `json:"details"   db:"details"`
}

const (
	kafkaBroker    = "localhost:9092"
	kafkaTopic     = "security-events"
	clickhouseAddr = "localhost:9000"
)

// setupKafkaWriter creates and returns a new Kafka writer.
func setupKafkaWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}
}

// setupClickHouseConnection connects to the ClickHouse database.
func setupClickHouseConnection() (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "s3cr3t",
		},
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func main() {
	// Initialize Kafka and ClickHouse connections
	kafkaWriter := setupKafkaWriter()
	defer kafkaWriter.Close()

	dbConn, err := setupClickHouseConnection()
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse for API: %v", err)
	}
	defer dbConn.Close()

	// Set up the Gin web server
	router := gin.Default()

	// CORS Middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Endpoint for receiving logs from our agents.
	router.POST("/ingest", func(c *gin.Context) {
		var event SecurityEvent
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		eventBytes, err := json.Marshal(event)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize event"})
			return
		}
		err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{Key: []byte(event.Hostname), Value: eventBytes})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish event"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "event received"})
	})

	// Endpoint for the dashboard
	router.GET("/events", func(c *gin.Context) {
		var events []SecurityEventForFrontend
		query := "SELECT timestamp, hostname, event_type, details FROM security_events ORDER BY timestamp DESC LIMIT 20"
		// The dbConn.Select method requires the struct tags to map the columns correctly.
		err := dbConn.Select(context.Background(), &events, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query events from database", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, events)
	})

	log.Println("Backend API server starting on port 8000...")
	router.Run(":8000")
}
