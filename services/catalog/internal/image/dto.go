package image

// ImageResponse is the shape a product image reaches a client in, whether
// answering a single upload or embedded in a product's image list. URL is
// presigned fresh for this response; ObjectKey never appears here.
type ImageResponse struct {
	ID       int64  `json:"id"`
	URL      string `json:"url"`
	Alt      string `json:"alt"`
	Position int    `json:"position"`
}

func newImageResponse(img Image, url string) ImageResponse {
	return ImageResponse{ID: img.ID, URL: url, Alt: img.Alt, Position: img.Position}
}
