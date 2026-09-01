package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
)

func toPostResponse(p *models.Post) dto.PostResponse {
	response := dto.PostResponse{
		ID:            p.ID,
		JournalID:     p.JournalID,
		Title:         ptrToStr(p.Title),
		Published:     p.Published,
		WrittenAt:     p.WrittenAt,
		CalendarType:  p.CalendarType,
		OriginalDay:   p.OriginalDay,
		OriginalMonth: p.OriginalMonth,
		OriginalYear:  p.OriginalYear,
		ViewCount:     p.ViewCount,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
	contacts := make([]dto.PostContactResponse, len(p.Contacts))
	for i, contact := range p.Contacts {
		contacts[i] = dto.PostContactResponse{
			ID:           contact.ID,
			FirstName:    ptrToStr(contact.FirstName),
			MiddleName:   ptrToStr(contact.MiddleName),
			LastName:     ptrToStr(contact.LastName),
			Nickname:     ptrToStr(contact.Nickname),
			MaidenName:   ptrToStr(contact.MaidenName),
			Prefix:       ptrToStr(contact.Prefix),
			Suffix:       ptrToStr(contact.Suffix),
			JobPosition:  ptrToStr(contact.JobPosition),
			LastTalkedTo: contact.LastTalkedTo,
		}
	}
	response.Contacts = contacts
	return response
}

func toPostResponseWithSections(p *models.Post) dto.PostResponse {
	resp := toPostResponse(p)
	sections := make([]dto.PostSectionResponse, len(p.PostSections))
	for i, section := range p.PostSections {
		sections[i] = dto.PostSectionResponse{
			ID:              section.ID,
			Position:        section.Position,
			Label:           section.Label,
			Content:         ptrToStr(section.Content),
			ContentFormat:   markdown.NormalizeFormat(section.ContentFormat),
			RenderedContent: markdown.Render(ptrToStr(section.Content), section.ContentFormat),
		}
	}
	resp.Sections = sections
	return resp
}
