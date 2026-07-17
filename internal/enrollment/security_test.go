package enrollment

import "testing"

func TestGeneratePairingTokenUsesUUIDv4(t *testing.T) {
	seen := map[string]struct{}{}
	for range 200 {
		token, err := GeneratePairingToken()
		if err != nil {
			t.Fatal(err)
		}
		if !LooksLikeUUID(token) || token[14] != '4' {
			t.Fatalf("unexpected UUID v4 %q", token)
		}
		if _, duplicate := seen[token]; duplicate {
			t.Fatalf("duplicate token generated: %s", token)
		}
		seen[token] = struct{}{}
	}
}

func TestAgentTokenRoundTrip(t *testing.T) {
	credentialID, secret, token, err := GenerateCredential()
	if err != nil {
		t.Fatal(err)
	}
	parsedID, parsedSecret, err := ParseAgentToken(token)
	if err != nil || parsedID != credentialID || parsedSecret != secret {
		t.Fatalf("round trip failed: id=%q secret=%q err=%v", parsedID, parsedSecret, err)
	}
	if HashSecret(secret) == secret || len(HashSecret(secret)) != 64 {
		t.Fatal("expected SHA-256 hash")
	}
}

func TestAgentTokenAllowsURLSafeUnderscoresInSecret(t *testing.T) {
	credentialID := "8b8f3bf4-5ea2-49e0-b07e-087f69973223"
	secret := "random_secret_with_underscores"
	parsedID, parsedSecret, err := ParseAgentToken(BuildAgentToken(credentialID, secret))
	if err != nil || parsedID != credentialID || parsedSecret != secret {
		t.Fatalf("underscore secret failed: id=%q secret=%q err=%v", parsedID, parsedSecret, err)
	}
}
