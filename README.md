# RDPIR - Ransomware Detection, Prevention and Incident Response

A comprehensive security monitoring system that provides ransomware detection, prevention and incident response capabilities using modern technologies.

## Architecture

The system consists of three main components:

- **Agent**: Python-based security event collector
- **Backend**: FastAPI service with Go consumer for event processing
- **Frontend**: React-based dashboard for monitoring and visualization

## Tech Stack

- **Backend**: Python (FastAPI), Go
- **Frontend**: React, Tailwind CSS
- **Message Queue**: Apache Kafka
- **Database**: ClickHouse
- **Containerization**: Docker & Docker Compose

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Python 3.8+
- Node.js 16+
- Go 1.19+

### Running with Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

### Manual Setup

1. **Start Infrastructure Services**
```bash
docker-compose up -d kafka clickhouse
```

2. **Backend Setup**
```bash
cd backend
pip install -r requirements.txt
python main.py
```

3. **Frontend Setup**
```bash
cd frontend
npm install
npm start
```

4. **Agent Setup**
```bash
cd agent
python agent.py
```

## API Endpoints

- `GET /health` - Health check
- `POST /ingest` - Ingest security events
- `GET /events` - Retrieve latest security events

## Configuration

### Environment Variables

- `KAFKA_BROKER`: Kafka broker address (default: localhost:9092)
- `CLICKHOUSE_HOST`: ClickHouse host (default: localhost)
- `CLICKHOUSE_PORT`: ClickHouse port (default: 8123)

## Development

### Project Structure

```
RDPIR/
├── agent/              # Security event collector
├── backend/            # API server and event consumer
├── frontend/           # React dashboard
├── docker-compose.yml  # Container orchestration
└── README.md
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

This project is licensed under the MIT License.