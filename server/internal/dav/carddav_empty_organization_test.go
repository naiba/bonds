package dav

import (
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
)

func TestCardDAVEmptyOrganizationStandaloneTitleHTTP(t *testing.T) {
	for _, org := range []string{"", ";", " ;Department"} {
		for _, title := range []string{"Architect", "", "   ", "omitted"} {
			for _, filtered := range []bool{false, true} {
				t.Run(fmt.Sprintf("org=%q/title=%q/filter=%v", org, title, filtered), func(t *testing.T) {
					e, db := setupDAVHTTPTestWithDB(t)
					user, email, password := createDAVHTTPTestUser(t, db)
					vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Artist")
					href := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id)
					request := func(method, body string) *httptest.ResponseRecorder {
						req := httptest.NewRequest(method, href, strings.NewReader(body))
						req.SetBasicAuth(email, password)
						req.Header.Set("Content-Type", "text/vcard")
						rec := httptest.NewRecorder()
						e.ServeHTTP(rec, req)
						return rec
					}
					original := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Artist\r\nN:Artist;Synthetic;;;\r\nTITLE:Artist\r\nEND:VCARD\r\n"
					if res := request("PUT", original); res.Code != 201 {
						t.Fatal(res.Code, res.Body.String())
					}
					if filtered {
						v := models.Vault{ID: vault, CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}
						if err := db.Model(&v).Select("CardDAVFilter").Updates(&v).Error; err != nil {
							t.Fatal(err)
						}
					}
					var before models.Contact
					if err := db.First(&before, "id = ?", id).Error; err != nil {
						t.Fatal(err)
					}
					previous := request("GET", "")
					input := strings.Replace(original, "TITLE:Artist", "ORG:"+org+"\r\nTITLE:"+title, 1)
					if title == "omitted" {
						input = strings.Replace(input, "TITLE:omitted\r\n", "", 1)
					}
					expected := strings.TrimSpace(title)
					if title == "omitted" {
						expected = ""
					}
					res := request("PUT", input)
					var c models.Contact
					if err := db.First(&c, "id = ?", id).Error; err != nil {
						t.Fatal(err)
					}
					if filtered && expected == "" {
						if res.Code != 409 {
							t.Errorf("clearing required standalone title returned %d, want 409", res.Code)
						}
					} else {
						if res.Code != 201 {
							t.Fatalf("valid standalone update failed: %d %s", res.Code, res.Body.String())
						}
						if got := ptrToStr(c.JobPosition); got != expected {
							t.Errorf("successful PUT ignored TITLE: stored=%q want=%q", got, expected)
						}
					}
					out := request("GET", "")
					if out.Code != 200 {
						t.Fatalf("GET after update: %d", out.Code)
					}
					if filtered && expected == "" {
						if !reflect.DeepEqual(before, c) || out.Body.String() != previous.Body.String() || out.Header().Get("ETag") != previous.Header().Get("ETag") {
							t.Error("rejected empty title changed actual persisted state or representation")
						}
					} else if expected == "" && (c.JobPosition != nil || strings.Contains(out.Body.String(), "TITLE:")) {
						t.Error("accepted empty title was not cleared to NULL")
					}
					for _, model := range []any{&models.Company{}, &models.ContactCompany{}} {
						var count int64
						if err := db.Model(model).Count(&count).Error; err != nil {
							t.Fatal(err)
						}
						if count != 0 {
							t.Error("empty ORG manufactured company or employment")
						}
					}
					t.Logf("PUT=%d database TITLE=%q GET=%d", res.Code, ptrToStr(c.JobPosition), out.Code)
					if !(filtered && expected == "") && expected != "" && !strings.Contains(out.Body.String(), "TITLE:"+expected+"\r\n") {
						t.Error("GET did not reflect accepted title")
					}
				})
			}
		}
	}
}

func TestCardDAVLegacyJobKeepsExistingTitleFilterVisible(t *testing.T) {
	for _, entry := range []string{"create job", "add employee", "existing mixed data"} {
		t.Run(entry, func(t *testing.T) {
			e, db := setupDAVHTTPTestWithDB(t)
			user, email, password := createDAVHTTPTestUser(t, db)
			vault, id := createDAVHTTPTestContact(t, db, user, "Synthetic", "Employee")
			old := models.Company{VaultID: vault, Name: "Original employer"}
			added := models.Company{VaultID: vault, Name: "Additional employer"}
			for _, c := range []*models.Company{&old, &added} {
				if err := db.Create(c).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": old.ID, "job_position": "Original title"}).Error; err != nil {
				t.Fatal(err)
			}
			v := models.Vault{ID: vault, CardDAVFilter: &models.CardDAVFilter{Match: "all", Fields: []string{"job_title"}}}
			if err := db.Model(&v).Select("CardDAVFilter").Updates(&v).Error; err != nil {
				t.Fatal(err)
			}
			read := func() *httptest.ResponseRecorder {
				req := httptest.NewRequest("GET", fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, id), nil)
				req.SetBasicAuth(email, password)
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				return rec
			}
			before := read()
			if before.Code != 200 || !strings.Contains(before.Body.String(), "TITLE:Original title") {
				t.Fatal("missing legacy fixture")
			}
			switch entry {
			case "create job":
				if _, err := services.NewContactJobService(db).Create(id, vault, dto.CreateContactJobRequest{CompanyID: added.ID}); err != nil {
					t.Fatal(err)
				}
			case "add employee":
				if _, err := services.NewContactJobService(db).AddEmployee(added.ID, vault, dto.AddEmployeeRequest{ContactID: id}); err != nil {
					t.Fatal(err)
				}
			case "existing mixed data":
				if err := db.Create(&models.ContactCompany{ContactID: id, CompanyID: added.ID}).Error; err != nil {
					t.Fatal(err)
				}
			}
			after := read()
			t.Logf("DAV status before=%d after adding unrelated titleless job=%d", before.Code, after.Code)
			if after.Code != 200 {
				t.Errorf("adding unrelated job hid a contact whose original title still exists: %d", after.Code)
			}
			if after.Code == 200 && (!strings.Contains(after.Body.String(), "TITLE:Original title") || !strings.Contains(after.Body.String(), "ORG:Original employer") || !strings.Contains(after.Body.String(), "ORG:Additional employer")) {
				t.Fatal("visible contact lost one of its employments")
			}
		})
	}
}
