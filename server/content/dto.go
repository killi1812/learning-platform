package content

import (
	"time"
)

// UpdateStaticPageDto defines metadata that can be updated for a static page / uploaded material.
type UpdateStaticPageDto struct {
	Title       string `json:"title" binding:"required,min=2,max=255"`
	Description string `json:"description,omitempty"`
}

// StaticPageDto represents the public view of course material stored in object storage.
type StaticPageDto struct {
	Uuid        string    `json:"uuid"`
	ModuleUuid  string    `json:"moduleUuid,omitempty"`
	Title       string    `json:"title"`
	Position    int       `json:"position"`
	FileName    string    `json:"fileName,omitempty"`
	FileSize    int64     `json:"fileSize,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
	Description string    `json:"description,omitempty"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// FromModel converts a StaticPage model into a StaticPageDto.
func (StaticPageDto) FromModel(sp *StaticPage) StaticPageDto {
	if sp == nil {
		return StaticPageDto{}
	}

	dto := StaticPageDto{
		FileName:    sp.FileName,
		FileSize:    sp.FileSize,
		ContentType: sp.ContentType,
		Description: sp.RichText,
		SourceURL:   sp.SourceURL,
	}

	if sp.Content != nil {
		dto.Uuid = sp.Content.Uuid.String()
		dto.Title = sp.Content.Title
		dto.Position = sp.Content.Position
		dto.CreatedAt = sp.Content.CreatedAt
		dto.UpdatedAt = sp.Content.UpdatedAt
		if sp.Content.Module != nil {
			dto.ModuleUuid = sp.Content.Module.Uuid.String()
		}
	}

	return dto
}
