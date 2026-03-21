# Gnosis Agent - Geochemical Decision Platform

A multi-tenant, agent-based geochemical decision platform that transforms raw assays into confident "Drill / Don't Drill" decisions.

## Features

- 🧪 **Multi-tenant Architecture**: Database-level isolation with separate MongoDB databases per organization
- 🤖 **Agent-Based Intelligence**: Go-ADK powered agents for QC, anomaly detection, prospectivity analysis, and decision orchestration
- 📊 **Advanced ML**: C++ mlpack integration via CGo for high-performance anomaly detection
- 📱 **Progressive Web App**: Offline-first UI with IndexedDB storage
- 📄 **Technical Reports**: LLM-powered automated report generation
- 🚀 **Single Executable**: Deploy as one binary with embedded web assets

## Quick Start

### Prerequisites

- Go 1.23+
- MongoDB 4.4+
- mlpack library with C++ compiler (for ML features)
- Templ CLI (for template generation)

### Installation

#### 1. Install dependencies

**macOS:**
```bash
brew install mlpack armadillo
go install github.com/a-h/templ/cmd/templ@latest
```

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y libmlpack-dev libarmadillo-dev g++
go install github.com/a-h/templ/cmd/templ@latest
```

#### 2. Clone and build

```bash
git clone https://gnosis-agent.git
cd geochem-agent

# Initialize Go modules
go mod download

# Build the application
chmod +x scripts/build.sh
./scripts/build.sh
```

#### 3. Configure

Create or edit `configs/org.yaml`:

```yaml
organization:
  id: "YOUR_ORG_ID"
  name: "Your Organization Name"
  deposit_models:
    - porphyry

mongodb:
  uri: "mongodb://localhost:27017"
  
llm:
  api_key: "your-openai-api-key"  # Or set GNOSISAGENT_LLM_API_KEY env var
```

#### 4. Run

```bash
./bin/GNOSISAGENT-$(go env GOOS)-$(go env GOARCH) start --config configs/org.yaml
```

Access the web interface at `http://localhost:8080`

## Project Structure

```
geochem-agent/
├── cmd/GNOSISAGENT/           # Main application entry point
├── internal/
│   ├── agents/             # Go-ADK agent implementations
│   ├── config/             # Configuration management
│   ├── db/                 # MongoDB with multi-tenancy
│   ├── handlers/           # Echo HTTP handlers
│   ├── middleware/         # HTTP middleware
│   ├── ml/mlpack/          # C++ mlpack CGo bridge
│   ├── models/             # Data models
│   └── services/           # Business logic services
├── web/
│   ├── static/             # Static assets (CSS, JS)
│   └── templates/          # Templ templates  
├── configs/                # Configuration files
├── scripts/                # Build and deployment scripts
└── tests/                  # Tests
```

## Architecture

### Database-Level Multi-Tenancy

Each organization gets a dedicated MongoDB database:
- Database naming: `GNOSISAGENT_org_{org_id}`
- Complete data isolation
- Independent backups and scaling

### Agent Pipeline

1. **QC Agent**: Validates data quality using statistical methods and mlpack anomaly detection
2. **Anomaly Detection Agent**: Identifies geochemical anomalies using Isolation Forest, LOF, and DBSCAN
3. **Prospectivity Agent**: Scores targets using ensemble ML + domain heuristics
4. **Sampling Optimization Agent**: Recommends optimal sample locations
5. **Decision Orchestrator**: Synthesizes agent outputs into drill decisions

### Technology Stack

- **Backend**: Go with Echo web framework
- **Database**: MongoDB (official driver)
- **Agents**: Latest Go-ADK from Google
- **ML**: C++ mlpack via CGo
- **Frontend**: Templ templates, IndexedDB
- **Reports**: OpenAI GPT-4 for technical writing

## Development

### Running tests

```bash
# Unit tests
go test ./...

# Integration tests (requires MongoDB)
docker run -d -p 27017:27017 mongo:latest
go test -tags=integration ./tests/integration/...
```

### Building Templ templates

```bash
templ generate
```

### Building with CGo

The mlpack integration requires CGo. Ensure you have:
- C++ compiler (g++ or clang)
- mlpack and armadillo libraries installed
- `CGO_ENABLED=1` environment variable

## Deployment

### Self-Hosted

```bash
# Build for your platform
./scripts/build.sh

# Deploy the binary
./bin/GNOSISAGENT-* start --config configs/production.yaml
```

### Docker

```bash
docker build -t GNOSISAGENT:latest .
docker run -p 8080:8080 -v ./configs:/configs GNOSISAGENT:latest start --config /configs/org.yaml
```

### Field Deployment

The single binary can run on laptops in remote locations with no  internet connection. All features work offline except LLM report generation.

## Configuration

See `configs/org.example.yaml` for all available options.

Key environment variables:
- `GNOSISAGENT_LLM_API_KEY`: OpenAI API key
- `GNOSISAGENT_MONGODB_URI`: MongoDB connection string
- `GNOSISAGENT_ORGANIZATION_ID`: Organization ID

## API Documentation

Coming soon...

## Contributing

Coming soon...

## License

MIT License

## Support

For issues and questions, please open a GitHub issue.
