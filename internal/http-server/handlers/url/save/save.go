package save

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"io"
	"log/slog"
	"net/http"
	resp "urlShortener/internal/lib/api/response"
	"urlShortener/internal/lib/logger/sl"
	"urlShortener/internal/lib/random"
	"urlShortener/internal/storage"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

// TODO: move to config
//
//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLSaver
const aliasLength = 6

type URLSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		err := render.DecodeJSON(r.Body, &req)
		if errors.Is(err, io.EOF) {
			// Такую ошибку встретим, если получили запрос с пустым телом.
			// Обработаем её отдельно
			log.Error("request body is empty")

			render.JSON(w, r, resp.Error("empty request"))

			return
		}
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))

			//render.JSON(w, r, resp.Error("invalid request"))
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		alias := req.Alias
		if alias != "" {
			id, err := urlSaver.SaveURL(req.URL, alias)
			if errors.Is(err, storage.ErrURLExists) {
				log.Info("url already exist", slog.String("url", req.URL))

				render.JSON(w, r, resp.Error("url already exist"))

				return
			}
			if err != nil {
				log.Info("failed to add url", slog.String("url", req.URL))

				render.JSON(w, r, resp.Error("failed to add url"))

				return
			}

			log.Info("url added", slog.Int64("id", id))

			responseOK(w, r, alias)
		} else {
			for {
				alias = random.NewRandomString(aliasLength)

				id, err := urlSaver.SaveURL(req.URL, alias)
				if errors.Is(err, storage.ErrURLExists) {
					continue
				}
				if err != nil {
					log.Info("failed to add url", slog.String("url", req.URL))

					render.JSON(w, r, resp.Error("failed to add url"))

					return
				}

				log.Info("url added", slog.Int64("id", id))

				responseOK(w, r, alias)

				return
			}
		}
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
