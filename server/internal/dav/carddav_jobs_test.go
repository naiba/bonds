package dav

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
)

func TestCardDAVJobsLifecycle(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
	contact := createTestContact(t, db, vaultID, userID, "Job", "Contact")
	book := fmt.Sprintf("/dav/addressbooks/%s/%s/", userID, vaultID)
	path := book + contact.ID + ".vcf"
	initial, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	company, err := services.NewCompanyService(db).Create(vaultID, dto.CreateCompanyRequest{Name: "Current company"})
	if err != nil {
		t.Fatal(err)
	}
	jobs := services.NewContactJobService(db)
	job, err := jobs.Create(contact.ID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	previousETag := initial.ETag
	check := func(org, title string, additional ...[2]string) {
		t.Helper()
		object, err := backend.GetAddressObject(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if object.ETag == previousETag {
			t.Fatal("job projection change did not change ETag")
		}
		previousETag = object.ETag
		listed, err := backend.ListAddressObjects(ctx, book, nil)
		if err != nil {
			t.Fatal(err)
		}
		queried, err := backend.QueryAddressObjects(ctx, book, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 || len(queried) != 1 {
			t.Fatalf("list/query returned %d/%d contacts", len(listed), len(queried))
		}
		for _, result := range []struct {
			card vcard.Card
			etag string
		}{{object.Card, object.ETag}, {listed[0].Card, listed[0].ETag}, {queried[0].Card, queried[0].ETag}} {
			var want [][2]string
			if org != "" || title != "" {
				want = append(want, [2]string{org, title})
			}
			want = append(want, additional...)
			if got := cardDAVJobPairs(result.card); !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected grouped jobs: got=%v want=%v", got, want)
			}
			if result.etag != object.ETag {
				t.Fatal("list/query/GET ETags disagree")
			}
			if result.card.Value(vcard.FieldVersion) != "3.0" {
				t.Fatal("CardDAV version changed")
			}
		}
	}
	check("Current company", "Engineer")
	if _, err := jobs.Update(contact.ID, vaultID, job.ID, dto.UpdateContactJobRequest{CompanyID: company.ID, JobPosition: "Manager"}); err != nil {
		t.Fatal(err)
	}
	check("Current company", "Manager")
	if _, err := services.NewCompanyService(db).Update(company.ID, vaultID, dto.UpdateCompanyRequest{Name: "Renamed company"}); err != nil {
		t.Fatal(err)
	}
	check("Renamed company", "Manager")
	if err := jobs.Delete(contact.ID, vaultID, job.ID); err != nil {
		t.Fatal(err)
	}
	check("", "")
	legacyCard, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	legacyCard.Card.SetValue(vcard.FieldTitle, "Remote legacy title")
	legacyUpdated, err := backend.PutAddressObject(ctx, path, legacyCard.Card, nil)
	if err != nil {
		t.Fatal(err)
	}
	if legacyUpdated.Card.Value(vcard.FieldTitle) != "Remote legacy title" {
		t.Fatal("legacy-only contact did not accept TITLE")
	}
	delete(legacyCard.Card, vcard.FieldTitle)
	legacyCleared, err := backend.PutAddressObject(ctx, path, legacyCard.Card, nil)
	if err != nil {
		t.Fatal(err)
	}
	if legacyCleared.Card.Value(vcard.FieldTitle) != "" {
		t.Fatal("PUT response retained a cleared legacy TITLE")
	}
	for _, title := range []string{"Older job", "Newest job"} {
		if _, err := jobs.Create(contact.ID, vaultID, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: title}); err != nil {
			t.Fatal(err)
		}
	}
	check("Renamed company", "Older job", [2]string{"Renamed company", "Newest job"})
	var before, after []models.ContactCompany
	if err := db.Order("id").Find(&before).Error; err != nil {
		t.Fatal(err)
	}
	object, err := backend.GetAddressObject(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	object.Card.SetName(&vcard.Name{GivenName: "Edited"})
	object.Card.SetValue(vcard.FieldFormattedName, "Edited")
	updated, err := backend.PutAddressObject(ctx, path, object.Card, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Card.Value(vcard.FieldFormattedName) != "Edited" || updated.Card.Value(vcard.FieldTitle) != "Older job" || updated.Card.Value(vcard.FieldOrganization) != "Renamed company" {
		t.Fatal("unrelated DAV edit lost name or job projection")
	}
	if err := db.Order("id").Find(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("DAV round trip modified local job history")
	}
	var stored models.Contact
	if err := db.First(&stored, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.JobPosition == nil || *stored.JobPosition != "Older job" || stored.PreferredJobID == nil || stored.CompanyID == nil || *stored.CompanyID != company.ID {
		t.Fatal("upstream employment selection was lost on the DAV round trip")
	}
	if !reflect.DeepEqual(cardDAVJobPairs(updated.Card), [][2]string{{"Renamed company", "Older job"}, {"Renamed company", "Newest job"}}) {
		t.Fatal("DAV round trip lost a grouped employment")
	}
	for _, job := range after {
		if err := jobs.Delete(contact.ID, vaultID, job.ID); err != nil {
			t.Fatal(err)
		}
	}
	check("", "")
}

func cardDAVJobPairs(card vcard.Card) [][2]string {
	var pairs [][2]string
	used := map[*vcard.Field]bool{}
	for _, org := range card[vcard.FieldOrganization] {
		name := ""
		components := org.Organization()
		if len(components) > 0 {
			name = components[0]
		}
		title := ""
		for _, field := range card[vcard.FieldTitle] {
			if field.Group == org.Group {
				title = field.Value
				used[field] = true
				break
			}
		}
		pairs = append(pairs, [2]string{name, title})
	}
	for _, field := range card[vcard.FieldTitle] {
		if !used[field] {
			pairs = append(pairs, [2]string{"", field.Value})
		}
	}
	return pairs
}
