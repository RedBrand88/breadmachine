#!/usr/bin/env bash
# Runs the handlers package tests that require the Firestore + Auth
# emulators (see handlers/emulator_helpers_test.go, handlers/recipe_emulator_test.go).
# These are skipped automatically by plain `go test ./...` — this script is
# the only way to actually execute them.
#
# Requires: Node/npm (uses `npx firebase-tools`, no global install needed)
# and a JDK on PATH (the emulators run on the JVM). Uses a "demo-*" project
# id, so no real GCP project or credentials are touched.
set -euo pipefail
cd "$(dirname "$0")/.."

npx --yes firebase-tools@15.28.1 emulators:exec \
  --project demo-breadmachine \
  --only firestore,auth \
  "go test ./handlers/... -run Emulator -v"
