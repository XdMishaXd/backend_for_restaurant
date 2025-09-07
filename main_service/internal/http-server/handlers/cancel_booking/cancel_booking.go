package cancelbooking

import (
	"errors"
	"log/slog"
	"main_service/internal/clients/sso/grpc"
	bookingsrv "main_service/internal/http-server/handlers/middleware/booking"
	resp "main_service/internal/lib/api/response"
	"main_service/internal/lib/logger/sl"
	"main_service/internal/models"
	"main_service/internal/storage"
	"main_service/internal/storage/postgres"
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

func New(
	log *slog.Logger,
	authClient *grpc.Client,
	bookingService *bookingsrv.BookingService,
	postgres *postgres.PostgresRepo,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.Cancel-booking.New"

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

		var req struct {
			TableID     int16     `json:"tableId" validate:"required"`
			BookingTime time.Time `json:"bookingTime" validate:"required"`
		}

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))

			render.JSON(w, r, resp.Error("Failed to decode request"))

			return
		}

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		if !isAdmin {
			ok, err := postgres.IsBookingOwner(r.Context(), req.TableID, req.BookingTime, int64(userID))
			if err != nil {
				log.Error("failed to check booking ownership", sl.Err(err))

				render.JSON(w, r, resp.Error("Failed to check booking ownership"))

				return
			}

			if !ok {
				log.Warn("user tried to cancel not his booking", slog.Int("userID", int(userID)))

				render.JSON(w, r, resp.Error("You can cancel only your own bookings"))

				return
			}
		}

		err = bookingService.CancelBooking(r.Context(), req.TableID, req.BookingTime)
		if err != nil {
			if errors.Is(err, storage.ErrBookingNotFound) {
				log.Warn("booking not found", slog.Int64("tableID", int64(req.TableID)), slog.Time("bookingTime", req.BookingTime))

				render.JSON(w, r, resp.Error("Booking not found"))

				return
			}

			log.Error("failed to cancel booking", slog.Any("err", err))

			render.JSON(w, r, resp.Error("Failed to cancel booking"))

			return
		}

		log.Info("booking canceled successfully",
			slog.Int("userID", int(userID)),
			slog.Int64("tableID", int64(req.TableID)),
			slog.Time("bookingTime", req.BookingTime),
		)

		ResponseOK(w, r)
	}
}

func ResponseOK(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Status:   "ok",
	})
}
