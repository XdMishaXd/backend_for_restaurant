package getbookings

import (
	"log/slog"
	"main_service/internal/clients/sso/grpc"
	bookingsrv "main_service/internal/http-server/handlers/middleware/booking"
	resp "main_service/internal/lib/api/response"
	"main_service/internal/lib/logger/sl"
	"main_service/internal/models"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
)

type GetBookingsRequest struct {
	Mode string `query:"mode" validate:"required,oneof=all active"`
}

func New(log *slog.Logger, authClient *grpc.Client, bookingService *bookingsrv.BookingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.book-table.GetBookings"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		req := GetBookingsRequest{
			Mode: r.URL.Query().Get("mode"),
		}

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

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

		if !isAdmin {
			log.Warn("customer attempted to view the reservation", slog.Int("userID", int(userID)))

			render.JSON(w, r, resp.Error("permisson denied"))

			return
		}

		// достаём список броней
		bookings, err := bookingService.GetBookings(r.Context(), req.Mode)
		if err != nil {
			log.Error("failed to get bookings", sl.Err(err))
			render.JSON(w, r, resp.Error("Failed to fetch bookings"))
			return
		}

		log.Info("bookings fetched successfully",
			slog.Int("count", len(bookings)),
			slog.String("mode", req.Mode),
		)

		render.JSON(w, r, resp.OKWithData(bookings))
	}
}
