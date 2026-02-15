package dto

type AddPostTagRequest struct {
	TagID uint   `json:"tag_id"`
	Name  string `json:"name"`
}

type UpdatePostTagRequest struct {
	Name string `json:"name" validate:"required"`
}

type PostTagResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
