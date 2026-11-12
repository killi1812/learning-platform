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

// CreateHandinDto defines the payload for creating a hand-in assignment.
type CreateHandinDto struct {
	Title                  string     `json:"title" binding:"required,min=2,max=255"`
	Deadline               *time.Time `json:"deadline,omitempty"`
	AcceptsLateSubmissions bool       `json:"acceptsLateSubmissions"`
	AllowsGroupSubmission  bool       `json:"allowsGroupSubmission"`
}

// UpdateHandinDto defines the payload for updating a hand-in assignment.
type UpdateHandinDto struct {
	Title                  string     `json:"title" binding:"required,min=2,max=255"`
	Deadline               *time.Time `json:"deadline,omitempty"`
	AcceptsLateSubmissions bool       `json:"acceptsLateSubmissions"`
	AllowsGroupSubmission  bool       `json:"allowsGroupSubmission"`
}

// HandinDto represents the public view of a hand-in assignment.
type HandinDto struct {
	Uuid                   string     `json:"uuid"`
	ModuleUuid             string     `json:"moduleUuid,omitempty"`
	Title                  string     `json:"title"`
	Position               int        `json:"position"`
	Deadline               *time.Time `json:"deadline,omitempty"`
	AcceptsLateSubmissions bool       `json:"acceptsLateSubmissions"`
	AllowsGroupSubmission  bool       `json:"allowsGroupSubmission"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

// FromModel converts Content + Assignment to HandinDto.
func (HandinDto) FromModel(cnt *Content) HandinDto {
	if cnt == nil {
		return HandinDto{}
	}
	dto := HandinDto{
		Uuid:      cnt.Uuid.String(),
		Title:     cnt.Title,
		Position:  cnt.Position,
		CreatedAt: cnt.CreatedAt,
		UpdatedAt: cnt.UpdatedAt,
	}
	if cnt.Module != nil {
		dto.ModuleUuid = cnt.Module.Uuid.String()
	}
	if cnt.Assignment != nil {
		dto.Deadline = cnt.Assignment.Deadline
		dto.AcceptsLateSubmissions = cnt.Assignment.AcceptsLateSubmissions
		dto.AllowsGroupSubmission = cnt.Assignment.AllowsGroupSubmission
	}
	return dto
}
