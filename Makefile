.PHONY: build run stop logs clean

# Build Docker image
build:
	@echo "🐳 Building Docker image..."
	docker build -t bottele-app .

# Run container
run:
	@echo "🚀 Starting container on port 1462..."
	docker stop bottele-telegram-bot 2>/dev/null || true
	docker rm bottele-telegram-bot 2>/dev/null || true
	docker run -d \
		--name bottele-telegram-bot \
		-p 1462:1462 \
		-v $(PWD)/data:/root/data \
		-v $(PWD)/grnstore.db:/root/grnstore.db \
		-e PORT=1462 \
		bottele-app
	@echo "✅ Container started! Check logs with: make logs"

# Stop container
stop:
	@echo "🛑 Stopping container..."
	docker stop bottele-telegram-bot

# View logs
logs:
	docker logs -f bottele-telegram-bot

# Clean up
clean:
	@echo "🧹 Cleaning up..."
	docker stop bottele-telegram-bot 2>/dev/null || true
	docker rm bottele-telegram-bot 2>/dev/null || true
	docker rmi bottele-app 2>/dev/null || true

# Build and run in one command
deploy: build run

# Quick restart
restart: stop run