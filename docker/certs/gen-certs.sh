#!/usr/bin/env bash
#
# Generate a self-signed CA plus server and client certificates for the TLS
# smoke tests. Idempotent: re-running regenerates everything. Portable across
# macOS and Linux (uses only openssl).
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

# The server cert is issued for CN=localhost with SANs so both hostname
# (verify-identity) and IP verification work when connecting to 127.0.0.1.
SUBJ_CA="/CN=dbdump-test-ca"
SUBJ_SERVER="/CN=localhost"
SUBJ_CLIENT="/CN=dbdump-client"
DAYS=3650

echo "Generating CA..."
openssl genrsa -out ca-key.pem 2048 2>/dev/null
openssl req -new -x509 -nodes -days "$DAYS" -key ca-key.pem -out ca.pem -subj "$SUBJ_CA" 2>/dev/null

echo "Generating server cert (CN=localhost, SAN localhost/127.0.0.1)..."
openssl genrsa -out server-key.pem 2048 2>/dev/null
openssl req -new -key server-key.pem -out server-req.pem -subj "$SUBJ_SERVER" 2>/dev/null
openssl x509 -req -in server-req.pem -days "$DAYS" \
  -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem \
  -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") 2>/dev/null

echo "Generating client cert (mutual TLS)..."
openssl genrsa -out client-key.pem 2048 2>/dev/null
openssl req -new -key client-key.pem -out client-req.pem -subj "$SUBJ_CLIENT" 2>/dev/null
openssl x509 -req -in client-req.pem -days "$DAYS" \
  -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out client-cert.pem 2>/dev/null

# MySQL requires its key files to be readable by the mysql user in-container and
# rejects world-writable keys; keep them 0644 (certs are test-only, not secret).
chmod 0644 ./*.pem
rm -f server-req.pem client-req.pem ca.srl

echo "Done. Files in $DIR:"
ls -1 ./*.pem
