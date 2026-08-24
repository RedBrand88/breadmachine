//go:build live

package handlers

import (
	"net/http"
	"testing"
)

// TestLive_FirebaseAdminV4_Smoke exercises the real (non-emulator) Firebase
// paths through the new firebase.google.com/go/v4 SDK to sanity-check the
// v3->v4 upgrade. It hits the real Firestore project and Google's real
// token-verification endpoint using the service account already present on
// this machine (/etc/breadmachine/serviceAccountKey.json) — no writes, no
// emulator. Run explicitly with: go test -tags=live ./handlers/...
func TestLive_FirebaseAdminV4_Smoke(t *testing.T) {
	if err := InitFirebase(); err != nil {
		t.Fatalf("InitFirebase failed under firebase.google.com/go/v4: %v", err)
	}

	recipes, err := FetchAllRecipesFromFirebase()
	if err != nil {
		t.Fatalf("FetchAllRecipesFromFirebase failed: %v", err)
	}
	if len(recipes) == 0 {
		t.Fatal("expected at least one recipe from the real Firestore project, got none")
	}

	req, err := http.NewRequest(http.MethodGet, "/api/recipes", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer not-a-real-token")

	// A garbage token must be rejected by a real round-trip to Google's
	// verification endpoint, not by a nil-pointer panic or SDK misbehavior
	// introduced by the import-path swap.
	if _, ok := authenticate(req); ok {
		t.Fatal("expected authenticate() to reject a garbage token, got ok=true")
	}
}
