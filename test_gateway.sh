#!/bin/bash

# Test script for API Gateway

echo "=== Testing API Gateway ==="
echo ""

# Test health check
echo "1. Testing Gateway Health Check:"
curl -s http://localhost:8080/health | jq .
echo ""

# Test backend status
echo "2. Testing Backend Status:"
curl -s http://localhost:8080/admin/backends | jq .
echo ""

# Test get all users (load balanced)
echo "3. Testing GET /api/users (should load balance):"
for i in {1..5}; do
  echo "Request $i:"
  curl -s http://localhost:8080/api/users | jq '.server'
done
echo ""

# Test get specific user
echo "4. Testing GET /api/users/1:"
curl -s http://localhost:8080/api/users/1 | jq .
echo ""

# Test create user
echo "5. Testing POST /api/users:"
curl -s -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Dave","email":"dave@example.com"}' | jq .
echo ""

# Test get orders
echo "6. Testing GET /api/orders:"
curl -s http://localhost:8080/api/orders | jq .
echo ""

# Test rate limiting (optional - sends many requests)
echo "7. Testing Rate Limiting (100 requests):"
echo "Sending 100 requests rapidly..."
success=0
limited=0
for i in {1..100}; do
  status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
  if [ "$status" == "200" ]; then
    ((success++))
  elif [ "$status" == "429" ]; then
    ((limited++))
  fi
done
echo "Success: $success, Rate Limited: $limited"
echo ""

echo "=== Tests Complete ==="
