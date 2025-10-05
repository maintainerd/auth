#!/bin/bash

# JWT Key Generation Script for Maintainerd Auth
# Generates RSA key pairs compliant with SOC2 and ISO27001 requirements
# 
# Usage: ./generate-jwt-keys.sh [key-size] [output-dir]
# Example: ./generate-jwt-keys.sh 4096 ./keys

set -euo pipefail

# Default values
DEFAULT_KEY_SIZE=4096
DEFAULT_OUTPUT_DIR="./keys"
MINIMUM_KEY_SIZE=2048

# Parse arguments
KEY_SIZE=${1:-$DEFAULT_KEY_SIZE}
OUTPUT_DIR=${2:-$DEFAULT_OUTPUT_DIR}

# Validate key size
if [ "$KEY_SIZE" -lt "$MINIMUM_KEY_SIZE" ]; then
    echo "❌ Error: Key size must be at least $MINIMUM_KEY_SIZE bits for security compliance"
    echo "   Recommended: 4096 bits for production environments"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "🔐 Generating JWT RSA Key Pair"
echo "   Key Size: $KEY_SIZE bits"
echo "   Output Directory: $OUTPUT_DIR"
echo ""

# Generate private key
echo "📝 Generating private key..."
openssl genrsa -out "$OUTPUT_DIR/jwt_private.pem" "$KEY_SIZE"

# Generate public key
echo "📝 Generating public key..."
openssl rsa -in "$OUTPUT_DIR/jwt_private.pem" -pubout -out "$OUTPUT_DIR/jwt_public.pem"

# Validate key pair
echo "🔍 Validating key pair..."
if openssl rsa -in "$OUTPUT_DIR/jwt_private.pem" -check -noout > /dev/null 2>&1; then
    echo "✅ Private key validation: PASSED"
else
    echo "❌ Private key validation: FAILED"
    exit 1
fi

# Generate environment variable format
echo "📋 Generating environment variable format..."

echo "# JWT Private Key (for .env file)" > "$OUTPUT_DIR/jwt_env_vars.txt"
echo -n "JWT_PRIVATE_KEY=\"" >> "$OUTPUT_DIR/jwt_env_vars.txt"
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' "$OUTPUT_DIR/jwt_private.pem" >> "$OUTPUT_DIR/jwt_env_vars.txt"
echo "\"" >> "$OUTPUT_DIR/jwt_env_vars.txt"

echo "" >> "$OUTPUT_DIR/jwt_env_vars.txt"
echo "# JWT Public Key (for .env file)" >> "$OUTPUT_DIR/jwt_env_vars.txt"
echo -n "JWT_PUBLIC_KEY=\"" >> "$OUTPUT_DIR/jwt_env_vars.txt"
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' "$OUTPUT_DIR/jwt_public.pem" >> "$OUTPUT_DIR/jwt_env_vars.txt"
echo "\"" >> "$OUTPUT_DIR/jwt_env_vars.txt"

# Generate key fingerprints for verification
echo "🔍 Generating key fingerprints..."
PRIVATE_FINGERPRINT=$(openssl rsa -in "$OUTPUT_DIR/jwt_private.pem" -pubout -outform DER | openssl dgst -sha256 -hex | cut -d' ' -f2)
PUBLIC_FINGERPRINT=$(openssl rsa -pubin -in "$OUTPUT_DIR/jwt_public.pem" -pubout -outform DER | openssl dgst -sha256 -hex | cut -d' ' -f2)

echo "Private Key Fingerprint: $PRIVATE_FINGERPRINT" > "$OUTPUT_DIR/key_fingerprints.txt"
echo "Public Key Fingerprint:  $PUBLIC_FINGERPRINT" >> "$OUTPUT_DIR/key_fingerprints.txt"

# Verify fingerprints match
if [ "$PRIVATE_FINGERPRINT" = "$PUBLIC_FINGERPRINT" ]; then
    echo "✅ Key pair fingerprint verification: PASSED"
else
    echo "❌ Key pair fingerprint verification: FAILED"
    exit 1
fi

# Set secure file permissions
chmod 600 "$OUTPUT_DIR/jwt_private.pem"
chmod 644 "$OUTPUT_DIR/jwt_public.pem"
chmod 600 "$OUTPUT_DIR/jwt_env_vars.txt"
chmod 644 "$OUTPUT_DIR/key_fingerprints.txt"

echo ""
echo "✅ JWT Key Generation Complete!"
echo ""
echo "📁 Generated Files:"
echo "   🔐 $OUTPUT_DIR/jwt_private.pem      - Private key (PEM format)"
echo "   🔓 $OUTPUT_DIR/jwt_public.pem       - Public key (PEM format)"
echo "   📋 $OUTPUT_DIR/jwt_env_vars.txt     - Environment variables"
echo "   🔍 $OUTPUT_DIR/key_fingerprints.txt - Key fingerprints"
echo ""
echo "🔒 Security Notes:"
echo "   • Private key permissions set to 600 (owner read/write only)"
echo "   • Store private key in secure key management system"
echo "   • Never commit keys to version control"
echo "   • Rotate keys every 90 days in production"
echo ""
echo "📋 Next Steps:"
echo "   1. Copy environment variables from jwt_env_vars.txt to your .env file"
echo "   2. Securely store the private key (consider using AWS KMS, HashiCorp Vault)"
echo "   3. Distribute public key to services that need to verify tokens"
echo "   4. Document key fingerprints for verification"
echo ""
echo "🧪 Test the keys:"
echo "   go run cmd/server/main.go"
echo ""
