# 🔐 Production Secret Management Guide

This guide covers secure secret management for production deployments of Maintainerd Auth, ensuring SOC2 and ISO27001 compliance.

## 🎯 **Secret Management Options**

### **1. Environment Variables (Development Only)**
```bash
# .env file (local development)
SECRET_PROVIDER=env
JWT_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
JWT_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
```

### **2. Docker/Kubernetes Secrets**
```bash
# Environment configuration
SECRET_PROVIDER=file
SECRET_FILE_PATH=/run/secrets

# Docker Swarm secrets
echo "your-private-key" | docker secret create jwt_private_key -
echo "your-public-key" | docker secret create jwt_public_key -

# Kubernetes secrets
kubectl create secret generic jwt-keys \
  --from-file=jwt_private_key=./keys/jwt_private.pem \
  --from-file=jwt_public_key=./keys/jwt_public.pem
```

### **3. AWS Systems Manager Parameter Store**
```bash
# Environment configuration
SECRET_PROVIDER=aws_ssm
SECRET_PREFIX=maintainerd/auth
AWS_REGION=us-east-1

# Store secrets in AWS SSM
aws ssm put-parameter \
  --name "/maintainerd/auth/JWT_PRIVATE_KEY" \
  --value "$(cat jwt_private.pem)" \
  --type "SecureString" \
  --description "JWT Private Key for Maintainerd Auth"

aws ssm put-parameter \
  --name "/maintainerd/auth/JWT_PUBLIC_KEY" \
  --value "$(cat jwt_public.pem)" \
  --type "String" \
  --description "JWT Public Key for Maintainerd Auth"
```

### **4. AWS Secrets Manager**
```bash
# Environment configuration
SECRET_PROVIDER=aws_secrets
SECRET_PREFIX=maintainerd/auth
AWS_REGION=us-east-1

# Store secrets in AWS Secrets Manager
aws secretsmanager create-secret \
  --name "maintainerd/auth/jwt-keys" \
  --description "JWT Keys for Maintainerd Auth" \
  --secret-string '{
    "JWT_PRIVATE_KEY": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
    "JWT_PUBLIC_KEY": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
  }'
```

### **5. HashiCorp Vault**
```bash
# Environment configuration
SECRET_PROVIDER=vault
SECRET_PREFIX=maintainerd/auth
VAULT_ADDR=https://vault.company.com
VAULT_TOKEN=your-vault-token

# Store secrets in Vault
vault kv put secret/maintainerd/auth/jwt-keys \
  JWT_PRIVATE_KEY="$(cat jwt_private.pem)" \
  JWT_PUBLIC_KEY="$(cat jwt_public.pem)"
```

---

## 🚀 **Production Deployment Examples**

### **Docker Compose with Secrets**
```yaml
version: '3.8'
services:
  auth:
    image: maintainerd/auth:latest
    environment:
      - SECRET_PROVIDER=file
      - SECRET_FILE_PATH=/run/secrets
    secrets:
      - jwt_private_key
      - jwt_public_key
    deploy:
      replicas: 3

secrets:
  jwt_private_key:
    external: true
  jwt_public_key:
    external: true
```

### **Kubernetes Deployment**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: maintainerd-auth
spec:
  replicas: 3
  selector:
    matchLabels:
      app: maintainerd-auth
  template:
    metadata:
      labels:
        app: maintainerd-auth
    spec:
      containers:
      - name: auth
        image: maintainerd/auth:latest
        env:
        - name: SECRET_PROVIDER
          value: "file"
        - name: SECRET_FILE_PATH
          value: "/etc/secrets"
        volumeMounts:
        - name: jwt-keys
          mountPath: "/etc/secrets"
          readOnly: true
      volumes:
      - name: jwt-keys
        secret:
          secretName: jwt-keys
```

### **AWS ECS with Parameter Store**
```json
{
  "family": "maintainerd-auth",
  "taskRoleArn": "arn:aws:iam::account:role/maintainerd-auth-task-role",
  "executionRoleArn": "arn:aws:iam::account:role/maintainerd-auth-execution-role",
  "containerDefinitions": [
    {
      "name": "auth",
      "image": "maintainerd/auth:latest",
      "environment": [
        {
          "name": "SECRET_PROVIDER",
          "value": "aws_ssm"
        },
        {
          "name": "SECRET_PREFIX",
          "value": "maintainerd/auth"
        },
        {
          "name": "AWS_REGION",
          "value": "us-east-1"
        }
      ]
    }
  ]
}
```

---

## 🔒 **Security Best Practices**

### **Key Generation**
```bash
# Generate production keys (4096-bit for enhanced security)
./scripts/generate-jwt-keys.sh 4096 ./production-keys

# Verify key strength
openssl rsa -in production-keys/jwt_private.pem -text -noout | grep "Private-Key"
```

### **Key Rotation Strategy**
1. **Generate new key pair** with new version ID
2. **Update secret storage** with new keys
3. **Deploy application** with new keys
4. **Verify token generation** works with new keys
5. **Monitor for validation errors** from old tokens
6. **Remove old keys** after grace period

### **Access Control**
```bash
# AWS IAM Policy for SSM Parameter Store
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameter",
        "ssm:GetParameters"
      ],
      "Resource": [
        "arn:aws:ssm:*:*:parameter/maintainerd/auth/*"
      ]
    }
  ]
}
```

### **Monitoring & Alerting**
```bash
# CloudWatch alarms for secret access
aws cloudwatch put-metric-alarm \
  --alarm-name "maintainerd-auth-secret-access-failures" \
  --alarm-description "Alert on secret access failures" \
  --metric-name "ErrorCount" \
  --namespace "AWS/SSM" \
  --statistic "Sum" \
  --period 300 \
  --threshold 5 \
  --comparison-operator "GreaterThanThreshold"
```

---

## 🧪 **Testing Secret Management**

### **Local Testing**
```bash
# Test with environment variables
SECRET_PROVIDER=env go run cmd/server/main.go

# Test with file secrets
mkdir -p /tmp/secrets
echo "$(cat jwt_private.pem)" > /tmp/secrets/jwt_private_key
echo "$(cat jwt_public.pem)" > /tmp/secrets/jwt_public_key
SECRET_PROVIDER=file SECRET_FILE_PATH=/tmp/secrets go run cmd/server/main.go
```

### **Production Validation**
```bash
# Verify secret loading
curl -H "Authorization: Bearer test-token" \
  https://auth.company.com/api/v1/health

# Check application logs
kubectl logs -f deployment/maintainerd-auth | grep "JWT keys loaded"
```

---

## 📋 **Compliance Checklist**

### **SOC2 Requirements**
- ✅ **CC6.1**: Secrets stored in secure, encrypted storage
- ✅ **CC6.8**: Key management with proper access controls
- ✅ **CC7.1**: Secrets transmitted over encrypted channels
- ✅ **CC8.1**: Change management for secret rotation

### **ISO27001 Requirements**
- ✅ **A.10.1.1**: Key management policy implemented
- ✅ **A.10.1.2**: Key lifecycle management
- ✅ **A.12.3.1**: Information backup (secret backup strategy)
- ✅ **A.13.2.1**: Information transfer (secure secret distribution)

### **Security Validation**
- ✅ Secrets never logged or exposed in error messages
- ✅ Secrets encrypted at rest and in transit
- ✅ Access to secrets requires proper authentication
- ✅ Secret access is audited and monitored
- ✅ Secrets are rotated regularly (90-day maximum)
- ✅ Old secrets are securely deleted after rotation

---

## 🚨 **Emergency Procedures**

### **Secret Compromise Response**
1. **Immediately rotate** compromised secrets
2. **Revoke all active tokens** (implement token blacklist)
3. **Update all deployments** with new secrets
4. **Monitor for unauthorized access** attempts
5. **Document incident** for compliance audit

### **Disaster Recovery**
1. **Backup secrets** in multiple secure locations
2. **Test secret restoration** procedures regularly
3. **Maintain secret recovery documentation**
4. **Verify backup integrity** monthly
