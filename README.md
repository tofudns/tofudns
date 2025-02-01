# Sample gRPC Service

This project demonstrates a simple gRPC service using Go, Chi, Buf, and Connect-Go.

## Prerequisites

- [mise](https://mise.jdx.dev/) - A dev tool manager

## Getting Started

1. Clone the repository:
   ```
   git clone https://github.com/tofudns/tofudns.git
   cd tofudns
   ```

2. Set up mise:
   ```
   mise trust
   mise install
   ```

3. Run the service:
   ```
   task compose:up
   task run
   ```
   The service will start and listen on port 8080.

4. Stop the service:
   ```
   task compose:down
   ```
   The service will stop and the containers will be removed.
