#!/bin/bash

# Define environments
ENVIRONMENTS=(
  "local=http://localhost:3006"
  "dev=https://dev.shopify-solver.ngrok.io"
  "prod=https://shopify-solver.ngrok.io"
)

# Default environment
ENV="local"

# Function to show usage
usage() {
  echo "Usage: $0 [options] ENDPOINT"
  echo ""
  echo "Options:"
  echo "  -e, --env ENV   Environment to use (local, dev, prod)"
  echo "  -m, --method M  HTTP method (GET, POST, PUT, DELETE)"
  echo "  -d, --data D    Request data (for POST/PUT)"
  echo "  -h, --help      Show this help"
  echo ""
  echo "Examples:"
  echo "  $0 /balances                    # GET request to local environment"
  echo "  $0 --env prod /balances         # GET request to production environment"
  echo "  $0 -e dev -m POST -d '{\"token\":\"SOL\",\"amount\":\"1.04\"}' /withdraw/token"
  echo ""
  exit 1
}

# Parse command-line arguments
METHOD="GET"
DATA=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -e|--env)
      ENV="$2"
      shift 2
      ;;
    -m|--method)
      METHOD="$2"
      shift 2
      ;;
    -d|--data)
      DATA="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    -*)
      echo "Unknown option: $1"
      usage
      ;;
    *)
      ENDPOINT="$1"
      shift
      ;;
  esac
done

# Validate environment
VALID_ENV=0
BASE_URL=""
for env_pair in "${ENVIRONMENTS[@]}"; do
  env_name="${env_pair%%=*}"
  env_url="${env_pair#*=}"
  if [[ "$ENV" == "$env_name" ]]; then
    BASE_URL="$env_url"
    VALID_ENV=1
    break
  fi
done

if [[ $VALID_ENV -eq 0 ]]; then
  echo "Error: Invalid environment '$ENV'"
  echo "Valid environments: local, dev, prod"
  exit 1
fi

# Validate endpoint
if [[ -z "$ENDPOINT" ]]; then
  echo "Error: No endpoint specified"
  usage
fi

# Ensure endpoint starts with /
if [[ "$ENDPOINT" != /* ]]; then
  ENDPOINT="/$ENDPOINT"
fi

# Full URL
URL="${BASE_URL}${ENDPOINT}"

# Make the request
echo "Making $METHOD request to $URL"

if [[ "$METHOD" == "GET" ]]; then
  curl -X GET "$URL" -H "accept: application/json" | jq '.'
elif [[ "$METHOD" == "POST" || "$METHOD" == "PUT" ]]; then
  if [[ -z "$DATA" ]]; then
    echo "Error: No data provided for $METHOD request"
    exit 1
  fi
  curl -X "$METHOD" "$URL" \
    -H "accept: application/json" \
    -H "Content-Type: application/json" \
    -d "$DATA" | jq '.'
else
  curl -X "$METHOD" "$URL" -H "accept: application/json" | jq '.'
fi 