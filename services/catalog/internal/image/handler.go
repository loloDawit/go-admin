package image

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/httpx"
)

// formFieldName is the multipart field POST .../images reads the image from.
const formFieldName = "file"

type Handler struct {
	svc      *Service
	maxBytes int64
	writeErr func(context.Context, http.ResponseWriter, error)
}

func NewHandler(svc *Service, maxBytes int64, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, maxBytes: maxBytes, writeErr: writeErr}
}

// Upload rejects on Content-Length before reading anything, then wraps the
// body in http.MaxBytesReader regardless, since Content-Length is a claim a
// client can lie about.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	productID, err := productIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	if r.ContentLength > h.maxBytes {
		h.writeErr(r.Context(), w, ErrImageTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)

	if err := r.ParseMultipartForm(h.maxBytes); err != nil {
		h.writeErr(r.Context(), w, httpx.ErrMalformedBody)
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile(formFieldName)
	if err != nil {
		h.writeErr(r.Context(), w, httpx.ErrMalformedBody)
		return
	}
	defer file.Close()

	created, err := h.svc.Upload(r.Context(), productID, File{
		Body:        file,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	url, err := h.svc.PresignURL(r.Context(), created.ObjectKey)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newImageResponse(created, url))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	productID, err := productIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	imageID, err := imageIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	if err := h.svc.Delete(r.Context(), productID, imageID); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func productIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}

func imageIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "imageID"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}
