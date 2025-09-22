package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/segmentio/kafka-go"
)

// This struct must match the one in our API server.
type SecurityEvent struct {
	Timestamp string `json:"timestamp"`
	Hostname  string `json:"hostname"`
	EventType string `json:"event_type"`
	Details   string `json:"details"`
}

const (
	kafkaBroker    = "localhost:9092"
	kafkaTopic     = "security-events"
	kafkaGroupID   = "sentinel-storage-group"
	clickhouseAddr = "localhost:9000"
)

// setupClickHouseConnection connects to ClickHouse using the native interface and ensures the table exists.
func setupClickHouseConnection() (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "s3cr3t", // Use the password set in docker-compose
		},
	})

	if err != nil {
		return nil, err
	}

	// Create the table if it doesn't already exist.
	const createTableQuery = `
	CREATE TABLE IF NOT EXISTS security_events (
		timestamp DateTime,
		hostname String,
		event_type String,
		details String
	) ENGINE = MergeTree()
	ORDER BY timestamp;
	`
	if err := conn.Exec(context.Background(), createTableQuery); err != nil {
		return nil, err
	}
	log.Println("ClickHouse table 'security_events' is ready.")
	return conn, nil
}

func main() {
	// Connect to ClickHouse
	conn, err := setupClickHouseConnection()
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer conn.Close()

	// Configure the Kafka reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		GroupID:  kafkaGroupID,
		Topic:    kafkaTopic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer r.Close()

	log.Println("Kafka consumer started. Waiting for events...")

	// Infinite loop to continuously read messages
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message from Kafka: %v", err)
			continue
		}

		var event SecurityEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}

		// Prepare to insert the event into ClickHouse
		batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO security_events")
		if err != nil {
			log.Printf("Failed to prepare batch for ClickHouse: %v", err)
			continue
		}

		// Parse the timestamp string into a time.Time object for ClickHouse
		ts, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			log.Printf("Failed to parse timestamp: %v", err)
			continue
		}

		if err := batch.Append(ts, event.Hostname, event.EventType, event.Details); err != nil {
			log.Printf("Failed to append to batch: %v", err)
			continue
		}

		// Send the batch to ClickHouse
		if err := batch.Send(); err != nil {
			log.Printf("Failed to send batch to ClickHouse: %v", err)
		} else {
			log.Printf("Successfully inserted event from host: %s", event.Hostname)
		}
	}
}
