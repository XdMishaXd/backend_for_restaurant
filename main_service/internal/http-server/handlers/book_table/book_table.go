package booktable

import (
	"errors"
	"log/slog"
	"main_service/internal/clients/sso/grpc"
	bookingsrv "main_service/internal/http-server/handlers/middleware/booking"
	resp "main_service/internal/lib/api/response"
	"main_service/internal/lib/logger/sl"
	"main_service/internal/models"
	"main_service/internal/storage"
	"net/http"

	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
)

type Request struct {
	TableID   int       `json:"tableId" validate:"required,gt=0"`
	BookingAt time.Time `json:"bookingAt" validate:"required"`
}

type Response struct {
	resp.Response
	Status string `json:"status"`
}

func New(log *slog.Logger, authClient *grpc.Client, bookingService *bookingsrv.BookingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.book-table.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, ok := r.Context().Value(models.ContextKey("uid")).(float64)
		if !ok || userID <= 0 {
			log.Error("unauthorized: no userID in context")

			render.JSON(w, r, resp.Error("Unauthorized"))

			return
		}

		isAdmin, err := authClient.IsAdmin(r.Context(), int64(userID))
		if err != nil {
			log.Error("failed to check user role", sl.Err(err))

			render.JSON(w, r, resp.Error("Failed to check user role"))

			return
		}

		if isAdmin {
			log.Warn("admin tried to book a table", slog.Int("userID", int(userID)))

			render.JSON(w, r, resp.Error("Admins cannot book tables"))

			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.JSON(w, r, resp.Error("Failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		// Валидация
		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		if time.Until(req.BookingAt) < 5*time.Hour {
			log.Warn("booking too close to current time", slog.Int("userID", int(userID)))

			render.JSON(w, r, resp.Error("You can only book at least 5 hours in advance"))

			return
		}

		booking := models.Booking{
			UserID:      int64(userID),
			TableID:     int16(req.TableID),
			BookingTime: req.BookingAt,
		}

		err = bookingService.BookTable(r.Context(), booking)
		if err != nil {
			if errors.Is(err, storage.ErrTableIsBooked) {
				log.Warn("failed to book table, table is already booked")

				render.JSON(w, r, resp.Error("Table is already booked"))

				return
			}

			log.Error("failed to book table", slog.Any("err", err))

			render.JSON(w, r, resp.Error("Failed to book table"))

			return
		}

		log.Info("table booked successfully", slog.Int("userID", int(userID)))

		ResponseOK(w, r)
	}
}

func ResponseOK(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Status:   "ok",
	})
}
