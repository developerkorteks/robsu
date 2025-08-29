#!/bin/bash

echo "🐳 Building Docker image for Bottele Telegram Bot..."
docker build -t bottele-app .

if [ $? -eq 0 ]; then
    echo "✅ Docker image built successfully!"
    echo "🚀 Starting container on port 1462..."
    
    # Stop existing container if running
    docker stop bottele-telegram-bot 2>/dev/null || true
    docker rm bottele-telegram-bot 2>/dev/null || true
    
    # Run new container
    docker run -d \
        --name bottele-telegram-bot \
        -p 1462:1462 \
        -v $(pwd)/data:/root/data \
        -v $(pwd)/grnstore.db:/root/grnstore.db \
        -e PORT=1462 \
        bottele-app
    
    if [ $? -eq 0 ]; then
        echo "✅ Container started successfully!"
        echo "📱 Bot is running on port 1462"
        echo "🔍 Check logs with: docker logs -f bottele-telegram-bot"
        echo "🛑 Stop with: docker stop bottele-telegram-bot"
    else
        echo "❌ Failed to start container"
    fi
else
    echo "❌ Failed to build Docker image"
fi