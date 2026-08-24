package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
)

var initEmulatorOnce sync.Once

// requireEmulators skips the test unless the Firestore and Auth emulators are
// running (FIRESTORE_EMULATOR_HOST / FIREBASE_AUTH_EMULATOR_HOST set, as
// `firebase emulators:exec` does for its child process). This keeps
// `go test ./...` hermetic — these tests only run under
// `scripts/test-emulators.sh`. On first call per test binary it also runs the
// real InitFirebase() once; with the emulator env vars already set, the
// resulting Firestore/Auth clients talk to the local emulators instead of
// production.
func requireEmulators(t *testing.T) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" || os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST / FIREBASE_AUTH_EMULATOR_HOST not set — run via scripts/test-emulators.sh")
	}
	initEmulatorOnce.Do(func() {
		if err := InitFirebase(); err != nil {
			t.Fatalf("InitFirebase against emulators failed: %v", err)
		}
	})
}

// mintIDToken returns a real ID token for the given uid against the Auth
// emulator, so tests can exercise authenticate()'s success path end-to-end
// rather than faking the token. The emulator's signUp endpoint doesn't accept
// a caller-chosen uid, so this goes through the standard emulator-testing
// pattern instead: mint a custom token for the uid via the Admin SDK (works
// identically against prod or emulator), then exchange it for an ID token via
// the emulator's signInWithCustomToken endpoint, which creates the user with
// that exact uid if it doesn't already exist.
func mintIDToken(t *testing.T, uid string) string {
	t.Helper()

	customToken, err := authClient.CustomToken(context.Background(), uid)
	if err != nil {
		t.Fatalf("failed to mint custom token: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"token":             customToken,
		"returnSecureToken": true,
	})

	url := fmt.Sprintf("http://%s/identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=fake-api-key",
		os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"))

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to reach auth emulator: %v", err)
	}
	defer resp.Body.Close()

	var out struct {
		IDToken string `json:"idToken"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode auth emulator response: %v", err)
	}
	if out.IDToken == "" {
		t.Fatalf("auth emulator signInWithCustomToken failed: %s", out.Error.Message)
	}
	return out.IDToken
}
