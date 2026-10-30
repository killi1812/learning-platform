package content

import (
	"time"
)

// UpdateFileDto defines metadata that can be updated for an uploaded file / course material.
type UpdateFileDto struct {
	Title       string `json:"title" binding:"required,min=2,max=255"`
	Description string `json:"description,omitempty"`
}

// FileDto represents the public view of course material stored in object storage.
type FileDto struct {
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

// FromModel converts a File model into a FileDto.
func (FileDto) FromModel(f *File) FileDto {
	if f == nil {
		return FileDto{}
	}

	dto := FileDto{
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		ContentType: f.ContentType,
		Description: f.RichText,
		SourceURL:   f.SourceURL,
	}

	if f.Content != nil {
		dto.Uuid = f.Content.Uuid.String()
		dto.Title = f.Content.Title
		dto.Position = f.Content.Position
		dto.CreatedAt = f.Content.CreatedAt
		dto.UpdatedAt = f.Content.UpdatedAt
		if f.Content.Module != nil {
			dto.ModuleUuid = f.Content.Module.Uuid.String()
		}
	}

	return dto
}
