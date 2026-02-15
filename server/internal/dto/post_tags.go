package dto

type AddPostTagRequest struct {
	TagID uint   `json:"tag_id" example:"1"`
	Name  string `json:"name" example:"Travel"`
}

type UpdatePostTagRequest struct {
	Name string `json:"name" validate:"required" example:"Travel"`
}

type PostTagResponse struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"Travel"`
	Slug string `json:"slug" example:"travel"`
}
