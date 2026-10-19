package dav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
)

func TestCardDAVStandaloneTitleRemoval(t *testing.T) {
	for _, titleField := range []string{"omitted", "empty", "blank"} {
		for _, filtered := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/filtered=%v", titleField, filtered), func(t *testing.T) {
				e, db := setupDAVHTTPTestWithDB(t)
				userID, email, password := createDAVHTTPTestUser(t, db)
				vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Synthetic", "Title")
				href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", userID, vaultID, contactID)
				request := func(method, body string) *httptest.ResponseRecorder {
					t.Helper()
					req := httptest.NewRequest(method, href, strings.NewReader(body))
					req.SetBasicAuth(email, password)
					req.Header.Set("Content-Type", "text/vcard")
					rec := httptest.NewRecorder()
					e.ServeHTTP(rec, req)
					return rec
				}
				original := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Title\r\nN:Title;Synthetic;;;\r\nTITLE:Independent consultant\r\nEND:VCARD\r\n"
				if rec := request(http.MethodPut, original); rec.Code != http.StatusCreated {
					t.Fatalf("initial PUT=%d: %s", rec.Code, rec.Body.String())
				}
				var before models.Contact
				if err := db.First(&before, "id = ?", contactID).Error; err != nil {
					t.Fatal(err)
				}
				if before.CompanyID != nil || ptrToStr(before.JobPosition) != "Independent consultant" {
					t.Fatal("invalid standalone title fixture")
				}
				note := models.Note{ContactID: contactID, VaultID: vaultID, Title: strPtrOrNil("Unrelated note"), Body: "Keep this private context"}
				if err := db.Create(&note).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&note, note.ID).Error; err != nil {
					t.Fatal(err)
				}
				if filtered {
					vault := models.Vault{ID: vaultID, CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}
					if err := db.Model(&vault).Select("CardDAVFilter").Updates(&vault).Error; err != nil {
						t.Fatal(err)
					}
				}
				previous := request(http.MethodGet, "")
				if previous.Code != http.StatusOK || !strings.Contains(previous.Body.String(), "TITLE:Independent consultant\r\n") {
					t.Fatal("initial title not readable")
				}
				replacement := ""
				if titleField == "empty" {
					replacement = "TITLE:\r\n"
				}
				if titleField == "blank" {
					replacement = "TITLE:   \r\n"
				}
				body := strings.Replace(previous.Body.String(), "TITLE:Independent consultant\r\n", replacement, 1)
				body = strings.ReplaceAll(body, "Synthetic", "Changed")
				result := request(http.MethodPut, body)
				afterGET := request(http.MethodGet, "")
				var after models.Contact
				if err := db.First(&after, "id = ?", contactID).Error; err != nil {
					t.Fatal(err)
				}
				var retainedNote models.Note
				if err := db.First(&retainedNote, note.ID).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(note, retainedNote) {
					t.Fatal("title edit changed an unrelated note")
				}
				for _, model := range []any{&models.Company{}, &models.ContactCompany{}} {
					var count int64
					if err := db.Model(model).Count(&count).Error; err != nil {
						t.Fatal(err)
					}
					if count != 0 {
						t.Fatal("standalone title manufactured employment records")
					}
				}
				if filtered {
					if result.Code != http.StatusConflict {
						t.Fatalf("filtered deletion=%d, want 409", result.Code)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatal("rejected deletion changed the stored contact")
					}
					if afterGET.Code != http.StatusOK || afterGET.Body.String() != previous.Body.String() || afterGET.Header().Get("ETag") != previous.Header().Get("ETag") {
						t.Fatal("rejected deletion changed DAV representation or ETag")
					}
					vault := models.Vault{ID: vaultID, CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{}}}
					if err := db.Model(&vault).Select("CardDAVFilter").Updates(&vault).Error; err != nil {
						t.Fatal(err)
					}
					if retry := request(http.MethodPut, body); retry.Code != http.StatusCreated {
						t.Fatalf("deletion after clearing filters=%d", retry.Code)
					}
					after = models.Contact{}
					if err := db.First(&after, "id = ?", contactID).Error; err != nil {
						t.Fatal(err)
					}
					afterGET = request(http.MethodGet, "")
				} else if result.Code != http.StatusCreated {
					t.Fatalf("deletion=%d: %s", result.Code, result.Body.String())
				}
				if after.JobPosition != nil || after.CompanyID != nil || ptrToStr(after.FirstName) != "Changed" {
					t.Fatal("accepted deletion was not persisted")
				}
				if afterGET.Code != http.StatusOK || strings.Contains(afterGET.Body.String(), "TITLE:") {
					t.Fatal("accepted deletion still exports TITLE")
				}
			})
		}
	}
}
