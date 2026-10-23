package dav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
)

func TestViewerCannotCreateEmploymentThroughDAV(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			userID, email, password := createDAVHTTPTestUser(t, db)
			vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Synthetic", "Viewer")
			if err := db.Model(&models.UserVault{}).Where("user_id = ? AND vault_id = ?", userID, vaultID).Update("permission", models.PermissionViewer).Error; err != nil {
				t.Fatal(err)
			}
			var before models.Contact
			if err := db.First(&before, "id = ?", contactID).Error; err != nil {
				t.Fatal(err)
			}
			href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contactID)
			request := func(method, path, body string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.SetBasicAuth(email, password)
				req.Header.Set("Content-Type", "text/vcard")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			if rec := request(http.MethodGet, href, ""); rec.Code != 200 {
				t.Fatalf("viewer lost read access: %d", rec.Code)
			}
			method := http.MethodPut
			if operation == "create" {
				href = fmt.Sprintf("/dav/addressbooks/%s/%s/new-employer.vcf", userID, vaultID)
			}
			if operation == "delete" {
				method = http.MethodDelete
			}
			rec := request(method, href, "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Unauthorized Change\r\nORG:Unauthorized employer\r\nTITLE:Engineer\r\nEND:VCARD\r\n")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("viewer %s=%d want 403", operation, rec.Code)
			}
			var companies, jobs, contacts int64
			if err := db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&companies).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.ContactCompany{}).Where("contact_id = ?", contactID).Count(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.Contact{}).Where("vault_id = ?", vaultID).Count(&contacts).Error; err != nil {
				t.Fatal(err)
			}
			if companies != 0 || jobs != 0 || contacts != 1 {
				t.Fatalf("rejected write changed records: companies=%d jobs=%d contacts=%d", companies, jobs, contacts)
			}
			var after models.Contact
			if err := db.First(&after, "id = ?", contactID).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(after.FirstName) != ptrToStr(before.FirstName) || !after.UpdatedAt.Equal(before.UpdatedAt) {
				t.Fatal("rejected write changed contact")
			}
		})
	}
}
