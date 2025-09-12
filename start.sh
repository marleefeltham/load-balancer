#!/bin/bash
#
# execute load balancer on config ports

#######################################
# configuration 
#######################################
BACKEND1_PORT=8081
BACKEND2_PORT=8082
BACKEND3_PORT=8083
LB_LOG="logs/loadbalancer.log"
BACKEND_LOG_DIR="logs"

# check that log dir exists
mkdir -p $BACKEND_LOG_DIR

#######################################
# start backends
#######################################
# backend 1: port 8081
echo "starting backend servers..."
export PORT=$BACKEND1_PORT
go run test_backends/main.go > "$BACKEND_LOG_DIR/backend1.log" 2>&1 &
BACKEND1_PID=$!
echo "Backend1 PID: $BACKEND1_PID (port $BACKEND1_PORT)"

# backend 2: port 8082
export PORT=$BACKEND2_PORT
go run test_backends/main.go > "$BACKEND_LOG_DIR/backend2.log" 2>&1 &
BACKEND2_PID=$!
echo "Backend2 PID: $BACKEND2_PID (port $BACKEND2_PORT)"

# backend 3: port 8083
export PORT=$BACKEND3_PORT
go run test_backends/main.go > "$BACKEND_LOG_DIR/backend3.log" 2>&1 &
BACKEND3_PID=$!
echo "Backend3 PID: $BACKEND3_PID (port $BACKEND3_PORT)"

# wait for backends to start
sleep 2

#######################################
# start load balancer
#######################################
echo "starting load balancer..."
go build -o load-balancer main.go
./load-balancer > "$LB_LOG" 2>&1 &
LB_PID=$!
echo "load balancer PID: $LB_PID (port from config.yaml)"

#######################################
# cleanup function
#######################################
cleanup() {
    echo "stopping load balancer and backends..."
    kill $LB_PID $BACKEND1_PID $BACKEND2_PID $BACKEND3_PID 2>/dev/null
    echo "stopped all processes."
    exit
}

# trap ctrl+c or termination signals
trap cleanup SIGINT SIGTERM

# wait for lb process to exit
wait $LB_PID
