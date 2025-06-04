#!/usr/bin/env bash
set -e

rm -rf bin || true
rm *.tgz || true

mkdir -p bin

# Build for Linux ARM64
GOOS=linux GOARCH=arm64 go build -o bin/homebridge-exporter-linux-arm64 .

# Build for Linux ARM (32-bit)
GOOS=linux GOARCH=arm go build -o bin/homebridge-exporter-linux-arm .

# Build for current platform (likely macOS ARM64)
go build -o bin/homebridge-exporter .

# Create tarballs
tar cvfz homebridge-exporter-linux-arm64.tgz bin/homebridge-exporter-linux-arm64
tar cvfz homebridge-exporter-linux-arm.tgz bin/homebridge-exporter-linux-arm
tar cvfz homebridge-exporter-darwin-arm64.tgz bin/homebridge-exporter
