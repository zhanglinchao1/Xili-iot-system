## Project Overview

XiLi is a **three-tier IoT energy storage cabinet monitoring system** with Zero-Knowledge Proof (ZKP) authentication. The repository contains three interconnected projects:

- **OrangePi (Gateway)**: IoT gateway collecting sensor data from physical devices via RS485/Modbus
- **Edge (Local Server)**: Middleware server providing ZKP authentication, local data storage, and cloud synchronization
- **Cloud (Platform)**: Central cloud platform for multi-cabinet management with web dashboard

**Key Technologies**: Go 1.24+, Gnark ZKP (Groth16), MQTT, PostgreSQL (Cloud), SQLite (Edge), Vue 3 + TypeScript (Cloud frontend), Vanilla JS (Edge frontend)

## System Architecture

```
Hardware Sensors (7 types)
  ↓ RS485/Modbus Serial
OrangePi Gateway (client2)
  ↓ MQTT TLS (port 8883) + ZKP Auth
Edge Server (github.com/edge/storage-cabinet)
  ↓ HTTP + MQTT TLS (port 8884)
Cloud Platform (cloud-system)
  ↓ Vue Dashboard
Web Users
