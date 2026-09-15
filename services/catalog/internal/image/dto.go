package image

import "strconv"

// ImageResponse is the shape a product image reaches a client in, whether
// answering a single upload or embedded in a product's image list. URL is
// presigned fresh for this response; ObjectKey never appears here.
type ImageResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Alt      string `json:"alt"`
	Position int    `json:"position"`
}

func newImageResponse(img Image, url string) ImageResponse {
	return ImageResponse{ID: strconv.FormatInt(img.ID, 10), URL: url, Alt: img.Alt, Position: img.Position}
}

type ListImagesResponse struct {
	Images []ImageResponse `json:"images"`
}
