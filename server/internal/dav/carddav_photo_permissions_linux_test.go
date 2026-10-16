package dav

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/naiba/bonds/pkg/avatar"
)

func TestCardDAVSearchableUploadDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		// Run the actual assertions without root's discretionary-access override.
		// Copy the test executable to a searchable path; Go's build directory may
		// be private to root. A non-root CI runner executes the assertions directly.
		directory, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Dir(directory), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0755); err != nil {
			t.Fatal(err)
		}
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		binary, err := os.ReadFile(executable)
		if err != nil {
			t.Fatal(err)
		}
		childBinary := filepath.Join(directory, "carddav-permissions.test")
		if err := os.WriteFile(childBinary, binary, 0755); err != nil {
			t.Fatal(err)
		}
		runtimeDirectory := filepath.Join(directory, "runtime")
		if err := os.Mkdir(runtimeDirectory, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chown(runtimeDirectory, 65534, 65534); err != nil {
			t.Fatal(err)
		}
		child := exec.Command(childBinary, "-test.run=^TestCardDAVSearchableUploadDirectory$", "-test.v")
		child.Dir = directory
		child.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + runtimeDirectory, "TMPDIR=" + runtimeDirectory}
		if dsn := os.Getenv(testutil.PostgresDSNEnv); dsn != "" {
			child.Env = append(child.Env, testutil.PostgresDSNEnv+"="+dsn)
		}
		child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
		output, err := child.CombinedOutput()
		t.Log(string(output))
		if err != nil {
			t.Fatalf("unprivileged photo regression: %v", err)
		}
		return
	}
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Private", "Directory")
	root := backend.uploadDir
	if err := os.Chmod(root, 0300); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0700); err != nil {
			t.Error(err)
		}
	})
	picture := avatar.GenerateInitials("Private directory", 32)
	uploaded, err := services.NewVaultFileService(db, root).UploadContactAvatar(contact.ID, vaultID, userID, "portrait.png", "image/png", int64(len(picture)), bytes.NewReader(picture))
	if err != nil {
		t.Fatalf("existing upload operation failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, uploaded.UUID))
	if err != nil || !bytes.Equal(data, picture) {
		t.Fatalf("existing known-file read failed: %v", err)
	}
	t.Log("non-root upload and direct UUID file read both succeeded with upload directory mode 0300")
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contact.ID)
	object, err := backend.GetAddressObject(ctx, href, nil)
	if err != nil {
		t.Errorf("CardDAV GET regressed for otherwise-readable selected photo: %v", err)
	} else {
		assertCardDAVPhoto(t, object.Card, picture, "PNG")
	}
	listed, err := backend.ListAddressObjects(ctx, fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one contact, got %d", len(listed))
	}
	assertCardDAVPhoto(t, listed[0].Card, picture, "PNG")
	card := vcard.Card{}
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetName(&vcard.Name{GivenName: "Edited", FamilyName: "Directory"})
	card.SetValue(vcard.FieldFormattedName, "Edited Directory")
	edited, err := backend.PutAddressObject(ctx, href, card, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCardDAVPhoto(t, edited.Card, picture, "PNG")
	if edited.Card.Value(vcard.FieldFormattedName) != "Edited Directory" {
		t.Fatal("ordinary CardDAV name edit was not applied")
	}
	if object != nil && edited.ETag == object.ETag {
		t.Fatal("name edit did not change ETag")
	}
	readback, err := backend.GetAddressObject(ctx, href, nil)
	if err != nil {
		t.Fatal(err)
	}
	if readback.Card.Value(vcard.FieldFormattedName) != "Edited Directory" || readback.ETag != edited.ETag {
		t.Fatal("name edit did not persist")
	}
	assertCardDAVPhoto(t, readback.Card, picture, "PNG")
}
