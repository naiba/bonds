package dav

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
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

// A merged standalone title remains independent of Jobs, including on an actual
// GET/PUT round trip and when deleting it is rejected by the Vault filter.
func TestCardDAVStandaloneTitleAlongsideJobs(t *testing.T) {
	e, db := setupDAVHTTPTestWithDB(t)
	user, email, password := createDAVHTTPTestUser(t, db)
	vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Independent")
	href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
	request := func(method, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, href, strings.NewReader(body))
		req.SetBasicAuth(email, password)
		req.Header.Set("Content-Type", "text/vcard")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	original := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Independent\r\nTITLE:Independent artist\r\nEND:VCARD\r\n"
	if rec := request(http.MethodPut, original); rec.Code != http.StatusCreated {
		t.Fatalf("create title: %d %s", rec.Code, rec.Body.String())
	}
	company := models.Company{VaultID: vault, Name: "Synthetic employer"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	job, err := services.NewContactJobService(db).Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID})
	if err != nil {
		t.Fatal(err)
	}
	get := request(http.MethodGet, "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), "employment1.TITLE:Independent artist\r\n") || !strings.Contains(get.Body.String(), "employment2.ORG:Synthetic employer\r\n") {
		t.Fatalf("mixed employment export: %d %s", get.Code, get.Body.String())
	}
	if rec := request(http.MethodPut, get.Body.String()); rec.Code != http.StatusCreated {
		t.Fatalf("round trip: %d %s", rec.Code, rec.Body.String())
	}
	var before models.Contact
	if err := db.First(&before, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if before.CompanyID != nil || before.PreferredJobID != nil || ptrToStr(before.JobPosition) != "Independent artist" {
		t.Fatal("round trip retargeted standalone selection")
	}
	filter := models.Vault{ID: vault, CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}
	if err := db.Model(&filter).Select("CardDAVFilter").Updates(&filter).Error; err != nil {
		t.Fatal(err)
	}
	get = request(http.MethodGet, "")
	body := strings.Replace(get.Body.String(), "employment1.TITLE:Independent artist\r\n", "", 1)
	if rec := request(http.MethodPut, body); rec.Code != http.StatusConflict {
		t.Fatalf("delete filtered title: %d %s", rec.Code, rec.Body.String())
	}
	var after models.Contact
	if err := db.First(&after, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected mixed employment write changed contact")
	}
	retained := request(http.MethodGet, "")
	if retained.Code != http.StatusOK || retained.Body.String() != get.Body.String() || retained.Header().Get("ETag") != get.Header().Get("ETag") {
		t.Fatal("rejected write changed representation")
	}
	filter.CardDAVFilter = &models.CardDAVFilter{Match: "all", Fields: []string{}}
	if err := db.Model(&filter).Select("CardDAVFilter").Updates(&filter).Error; err != nil {
		t.Fatal(err)
	}
	if rec := request(http.MethodPut, body); rec.Code != http.StatusCreated {
		t.Fatalf("unfiltered title deletion: %d %s", rec.Code, rec.Body.String())
	}
	if err := db.First(&after, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if after.JobPosition != nil {
		t.Fatal("deleted standalone title remains persisted")
	}
	remaining, err := services.NewContactJobService(db).List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || !reflect.DeepEqual(remaining[0], *job) {
		t.Fatal("round trip or title removal changed unrelated Jobs")
	}
	if rec := request(http.MethodGet, ""); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "TITLE:") || !strings.Contains(rec.Body.String(), "ORG:Synthetic employer") {
		t.Fatal("title removal lost company or still exports standalone title")
	}
}
