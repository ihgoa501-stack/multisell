package auth

import "testing"

func TestExtensionPairingOwnerFlow(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ExtensionPairing{}, &ExtensionDevice{}); err != nil {
		t.Fatal(err)
	}
	owner := User{Username: "pair-owner", PasswordHash: "unused", Role: "owner", Status: 1}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, testConfig(), testLogger())
	created, err := svc.CreateExtensionPairing(owner.ID, "development")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ClaimExtensionPairing(created.Nonce, "claim-secret", "install-1", "extension-1", "development", "Chrome"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmExtensionPairing(owner.ID, created.PairingID); err != nil {
		t.Fatal(err)
	}
	credential, err := svc.ExchangeExtensionPairing(created.Nonce, "claim-secret")
	if err != nil || credential.AccessToken == "" || credential.DeviceSecret == "" {
		t.Fatalf("credential=%+v err=%v", credential, err)
	}
	if _, err := svc.RefreshExtensionDevice(credential.DeviceID, credential.DeviceSecret, "development"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeExtensionDevice(owner.ID, credential.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RefreshExtensionDevice(credential.DeviceID, credential.DeviceSecret, "development"); err == nil {
		t.Fatal("revoked device refreshed")
	}
}
