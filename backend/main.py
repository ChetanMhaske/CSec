import json
import logging
from datetime import datetime

import clickhouse_connect
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from kafka import KafkaProducer
from pydantic import BaseModel

# --- Configuration ---
KAFKA_BROKER = "localhost:9092"
KAFKA_TOPIC = "security-events"
CLICKHOUSE_HOST = "localhost"
CLICKHOUSE_PORT = 8123
CLICKHOUSE_USER = "default"
CLICKHOUSE_PASSWORD = "s3cr3t"

# --- Logging Setup ---
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# --- Pydantic Models ---
class SecurityEvent(BaseModel):
    timestamp: str
    hostname: str
    event_type: str
    details: str

class SecurityEventForFrontend(BaseModel):
    timestamp: datetime
    hostname: str
    event_type: str
    details: str

    class Config:
        from_attributes = True

# --- FastAPI App Initialization ---
app = FastAPI(title="Sentinel AI Backend")

# --- CORS Middleware ---
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000"],  # Allow your frontend origin
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# --- Kafka Producer Setup ---
producer = None
try:
    producer = KafkaProducer(
        bootstrap_servers=KAFKA_BROKER,
        value_serializer=lambda v: json.dumps(v).encode('utf-8'),
        api_version=(0, 10, 1) # Specify a compatible API version
    )
    logger.info("Successfully connected to Kafka.")
except Exception as e:
    logger.error(f"Failed to connect to Kafka: {e}")

# --- ClickHouse Client Setup ---
client = None
try:
    client = clickhouse_connect.get_client(
        host=CLICKHOUSE_HOST,
        port=CLICKHOUSE_PORT,
        user=CLICKHOUSE_USER,
        password=CLICKHOUSE_PASSWORD
    )
    logger.info("Successfully connected to ClickHouse.")
except Exception as e:
    logger.error(f"Failed to connect to ClickHouse: {e}")

# --- API Endpoints ---
@app.get("/health", tags=["Monitoring"])
def health_check():
    """Check if the API is running."""
    return {"status": "UP"}

@app.post("/ingest", status_code=200, tags=["Agent"])
def ingest_event(event: SecurityEvent):
    """Receives a security event from an agent and publishes it to Kafka."""
    if not producer:
        raise HTTPException(status_code=500, detail="Kafka producer is not available.")
    try:
        producer.send(KAFKA_TOPIC, event.dict())
        logger.info(f"Event from {event.hostname} sent to Kafka.")
        return {"status": "event received"}
    except Exception as e:
        logger.error(f"Failed to publish event to Kafka: {e}")
        raise HTTPException(status_code=500, detail="Failed to publish event.")

@app.get("/events", response_model=list[SecurityEventForFrontend], tags=["Dashboard"])
def get_events():
    """Retrieves the latest 20 security events from the database for the dashboard."""
    global client
    if not client or not client.ping():
        try:
            logger.info("ClickHouse client is not active. Attempting to reconnect...")
            client = clickhouse_connect.get_client(
                host=CLICKHOUSE_HOST,
                port=CLICKHOUSE_PORT,
                user=CLICKHOUSE_USER,
                password=CLICKHOUSE_PASSWORD
            )
            logger.info("Successfully reconnected to ClickHouse.")
        except Exception as e:
            logger.error(f"Failed to reconnect to ClickHouse: {e}")
            raise HTTPException(status_code=500, detail=f"ClickHouse connection failed: {str(e)}")

    try:
        query = "SELECT timestamp, hostname, event_type, details FROM security_events ORDER BY timestamp DESC LIMIT 20"
        result = client.query(query)
        events = [
            SecurityEventForFrontend(
                timestamp=row[0],
                hostname=row[1],
                event_type=row[2],
                details=row[3]
            )
            for row in result.result_rows
        ]
        return events
    except Exception as e:
        logger.error(f"Failed to query events from database: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to query events from database: {str(e)}")

# --- Uvicorn Runner ---
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)