# GeoAgent Platform - Deployment Guide

## Quick Start

### Prerequisites
- Docker & Docker Compose (recommended)
- OR Go 1.21+, MongoDB, and mlpack (for local development)

### Option 1: Docker Deployment (Recommended)

1. **Clone and Configure**
   ```bash
   git clone https://github.com/pipdaniels/geochem-agent.git
   cd geochem-agent
   
   # Copy and configure environment
   cp .env.example .env
   # Edit .env with your Gemini API key and passwords
   ```

2. **Start Services**
   ```bash
   docker-compose up -d
   ```

3. **Access Application**
   - GeoAgent Platform: http://localhost:8080
   - MongoDB Admin (optional): http://localhost:8081

4. **Check Health**
   ```bash
   curl http://localhost:8080/health
   ```

### Option 2: Binary Deployment

1. **Download Pre-built Binary**
   ```bash
   # Download for your platform from releases
   wget https://github.com/pipdaniels/geochem-agent/releases/download/v1.0.0/geoagent-linux-amd64
   chmod +x geoagent-linux-amd64
   ```

2. **Setup MongoDB**
   ```bash
   # Using Docker
   docker run -d -p 27017:27017 --name mongodb mongo:7
   
   # Or install locally
   # macOS:  brew install mongodb-community
   # Ubuntu: sudo apt-get install mongodb-org
   ```

3. **Configure**
   ```bash
   cp configs/org.example.yaml configs/org.yaml
   # Edit configs/org.yaml with your settings
   export GEOAGENT_LLM_API_KEY=your_gemini_api_key
   ```

4. **Run**
   ```bash
   ./geoagent-linux-amd64 start --config configs/org.yaml
   ```

### Option 3: Build from Source

1. **Install Dependencies**
   ```bash
   # macOS
   brew install mlpack armadillo go
   
   # Ubuntu/Debian
   sudo apt-get install libmlpack-dev libarmadillo-dev golang
   
   # Install Templ
   go install github.com/a-h/templ/cmd/templ@latest
   ```

2. **Build**
   ```bash
   ./scripts/build.sh
   ```

3. **Run**
   ```bash
   ./bin/geoagent-$(go env GOOS)-$(go env GOARCH) --config configs/org.yaml
   ```

## Production Deployment

### Docker Swarm

```yaml
# docker-stack.yml
version: '3.8'

services:
  geoagent:
    image: geoagent:latest
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
    ports:
      - "8080:8080"
    environment:
      - GEOAGENT_MONGODB_URI=mongodb://mongodb:27017
    networks:
      - geoagent-net

  mongodb:
    image: mongo:7
    deploy:
      placement:
        constraints: [node.role == manager]
    volumes:
      - mongodb-data:/data/db
    networks:
      - geoagent-net

volumes:
  mongodb-data:

networks:
  geoagent-net:
    driver: overlay
```

Deploy:
```bash
docker stack deploy -c docker-stack.yml geoagent
```

### Kubernetes

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: geoagent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: geoagent
  template:
    metadata:
      labels:
        app: geoagent
    spec:
      containers:
      - name: geoagent
        image: geoagent:latest
        ports:
        - containerPort: 8080
        env:
        - name: GEOAGENT_MONGODB_URI
          valueFrom:
            secretKeyRef:
              name: geoagent-secrets
              key: mongodb-uri
        - name: GEOAGENT_LLM_API_KEY
          valueFrom:
            secretKeyRef:
              name: geoagent-secrets
              key: gemini-api-key
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: geoagent
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: geoagent
```

Deploy:
```bash
kubectl apply -f k8s/
```

### Systemd Service (Linux)

```ini
# /etc/systemd/system/geoagent.service
[Unit]
Description=GeoAgent Geochemical Platform
After=network.target mongodb.service

[Service]
Type=simple
User=geoagent
WorkingDirectory=/opt/geoagent
ExecStart=/opt/geoagent/bin/geoagent start --config /opt/geoagent/config/org.yaml
Restart=always
RestartSec=10
Environment="GEOAGENT_LLM_API_KEY=your_api_key"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable geoagent
sudo systemctl start geoagent
sudo systemctl status geoagent
```

## Configuration

### Environment Variables

- `GEOAGENT_MONGODB_URI`: MongoDB connection string
- `GEOAGENT_LLM_API_KEY`: Google Gemini API key
- `GEOAGENT_SERVER_PORT`: Server port (default: 8080)
- `GEOAGENT_ORG_ID`: Organization ID
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

### Config File (configs/org.yaml)

See `configs/org.example.yaml` for full configuration options.

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "org": "demo_org"
}
```

### Logs

Docker:
```bash
docker-compose logs -f geoagent
```

Binary:
```bash
journalctl -u geoagent -f
```

## Scaling

### Horizontal Scaling

The application is stateless and can be scaled horizontally:

Docker Compose:
```bash
docker-compose up -d --scale geoagent=3
```

Kubernetes:
```bash
kubectl scale deployment geoagent --replicas=5
```

### Database Scaling

For multi-tenant deployments with many organizations, consider:
- MongoDB sharding
- Read replicas for analytics
- Separate clusters per region

## Security

### Production Checklist

- [ ] Change all default passwords
- [ ] Enable HTTPS/TLS
- [ ] Configure firewall rules
- [ ] Set up authentication (JWT)
- [ ] Use secrets management (Vault, AWS Secrets Manager)
- [ ] Enable MongoDB authentication
- [ ] Regular backups
- [ ] Security updates

### HTTPS Setup

Use a reverse proxy (nginx, Caddy, Traefik):

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name geoagent.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Backup & Restore

### MongoDB Backup

```bash
# Backup
docker exec geoagent-mongodb mongodump --out /backup

# Restore
docker exec geoagent-mongodb mongorestore /backup
```

### Automated Backups

```bash
# Cron job (daily at 2 AM)
0 2 * * * docker exec geoagent-mongodb mongodump --out /backup/$(date +\%Y\%m\%d)
```

## Troubleshooting

### Cannot Connect to MongoDB

```bash
# Check MongoDB is running
docker-compose ps mongodb

# Check logs
docker-compose logs mongodb

# Test connection
docker exec -it geoagent-mongodb mongosh
```

### Application Won't Start

```bash
# Check logs
docker-compose logs geoagent

# Verify configuration
docker-compose config

# Check health
curl http://localhost:8080/health
```

### MLPack Issues

If MLPack is not available, the application will run without ML-powered anomaly detection. To enable:

```bash
# Install mlpack
brew install mlpack armadillo  # macOS
apt-get install libmlpack-dev  # Ubuntu

# Rebuild with CGo
./scripts/build.sh
```

## Performance Tuning

### Go Application

- Increase `GOMAXPROCS` for multi-core systems
- Tune garbage collection: `GOGC=100` (default)
- Use profiling: `pprof` endpoints

### MongoDB

```javascript
// Indexes for performance
db.qc_results.createIndex({"dataset_id": 1})
db.anomaly_results.createIndex({"dataset_id": 1})
db.decisions.createIndex({"created_at": -1})
```

### Docker

```yaml
# docker-compose.yml
services:
  geoagent:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

## Support

- Documentation: https://geoagent.docs
- Issues: https://github.com/pipdaniels/geochem-agent/issues
- Email: support@geoagent.com
