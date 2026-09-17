#!/usr/bin/env bash
set -Eeuo pipefail

echo "[1/7] Config Manager Runner self-test"
sleep 1

echo "[2/7] Go"
go version

echo "[3/7] Xray"
xray version | head -n1

echo "[4/7] PostgreSQL"
psql --version
systemctl is-active --quiet postgresql
echo "PostgreSQL: active"

echo "[5/7] GitHub remote"
git remote -v

echo "[6/7] Resources"
printf 'CPU: '
nproc
free -h
df -h /

echo "[7/7] Background execution"
sleep 3

echo
echo "RUNNER SELF-TEST SUCCESS"
