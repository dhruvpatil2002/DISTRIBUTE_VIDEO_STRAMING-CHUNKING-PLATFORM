package routes

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/auth"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/config"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/handlers"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/models"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/repository"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/stream"
)

func New(
	db *pgxpool.Pool,
	cfg *config.Config,
) (http.Handler, func(), error) {

	mux := http.NewServeMux()

	// =========================
	// Repositories
	// =========================

	userRepo := repository.NewUserRepository(db)
	siteRepo := repository.NewSiteRepository(db)
	cameraRepo := repository.NewCameraRepository(db)

	// =========================
	// Services
	// =========================

	userService := service.NewUserService(
		userRepo,
		cfg.JWTSecret,
	)

	siteService := service.NewSiteService(siteRepo)

	// =========================
	// Authentication Middleware
	// =========================

	authMiddleware := auth.Middleware(userService)

	// =========================
	// Stream Managers
	// =========================

	log.Printf(
		"MediaMTX: API=%s, userConfigured=%t, passwordConfigured=%t",
		cfg.MediaMTXAPI,
		cfg.MediaMTXAPIUser != "",
		cfg.MediaMTXAPIPassword != "",
	)

	rtspManager := stream.NewMediaMTXManager(
		cfg.MediaMTXAPI,
		cfg.MediaMTXRTSP,
		cfg.MediaMTXAPIUser,
		cfg.MediaMTXAPIPassword,
	)

	hlsManager := stream.NewHLSManager(
		cfg.FFmpegPath,
		cfg.HLSDir,
		cfg.StreamGracePeriod,
		time.Minute,
		2*time.Second,
		5,
	)

	// =========================
	// Restore Registered Cameras
	// =========================
// =========================
// Restore Registered Cameras
// =========================

ctx, cancel := context.WithTimeout(
    context.Background(),
    30*time.Second,
)
defer cancel()

cameras, err := cameraRepo.GetAllForStreaming(ctx)
if err != nil {
    hlsManager.Shutdown()

    return nil, nil, fmt.Errorf(
        "load registered cameras: %w",
        err,
    )
}

for _, camera := range cameras {
    log.Printf(
        "Restoring camera %s into MediaMTX",
        camera.ID,
    )

    err := rtspManager.EnsureCamera(
        ctx,
        camera.ID,
        camera.RTSPURL,
    )

    if err != nil {
        hlsManager.Shutdown()

        return nil, nil, fmt.Errorf(
            "restore camera %s: %w",
            camera.ID,
            err,
        )
    }

    log.Printf(
        "Camera %s synchronized with MediaMTX",
        camera.ID,
    )
}

log.Printf(
    "Camera restoration completed: %d cameras",
    len(cameras),
)

	// =========================
	// Camera Service
	// =========================

	cameraService := service.NewCameraService(
		cameraRepo,
		rtspManager,
		hlsManager,
	)

	// =========================
	// Handlers
	// =========================

	userHandler := handlers.NewUserHandler(userService)

	siteHandler := handlers.NewSiteHandler(
		siteService,
	)

	cameraHandler := handlers.NewCameraHandler(
		cameraService,
	)

	playbackHandler := handlers.NewPlaybackHandler(
		cameraService,
	)

	// =========================
	// Public Routes
	// =========================

	mux.HandleFunc(
		"GET /health",
		handlers.Health(),
	)

	// User Registration
	mux.HandleFunc(
		"POST /api/auth/register",
		userHandler.Register,
	)

	// User Login
	mux.HandleFunc(
		"POST /api/auth/login",
		userHandler.Login,
	)

	// =========================
	// Protected Authentication Routes
	// =========================

	protected := func(h http.Handler) http.Handler {
		return authMiddleware(h)
	}

	adminOnly := func(h http.Handler) http.Handler {
		return authMiddleware(
			auth.RequireRole("admin")(h),
		)
	}

	// =========================
	// Protected Site Routes
	// =========================
	mux.Handle(
		 "GET /api/admin/users",
    auth.Middleware(userService)(
        auth.RequireRole(models.RoleAdmin)(
            http.HandlerFunc(userHandler.GetAllUsers),),
		),
	)


	mux.Handle(
    "PATCH /api/admin/users/{id}/role",
    auth.Middleware(userService)(
        auth.RequireRole(models.RoleAdmin)(
            http.HandlerFunc(userHandler.UpdateRole),
        ),
    ),
)
	mux.Handle(
		"POST /api/sites",
		adminOnly(http.HandlerFunc(siteHandler.Create)),
	)

	mux.Handle(
		"GET /api/sites/{siteID}",
		protected(http.HandlerFunc(siteHandler.GetByID)),
	)

	mux.Handle(
		"PUT /api/sites/{siteID}",
		adminOnly(http.HandlerFunc(siteHandler.Update)),
	)

	mux.Handle(
		"DELETE /api/sites/{siteID}",
		adminOnly(http.HandlerFunc(siteHandler.Delete)),
	)
	mux.Handle(
		"GET /api/sites/getAll",
		protected(http.HandlerFunc(siteHandler.GetAll)),
	)

	// =========================
	// Protected Camera Routes
	// =========================

	// =========================
	// Camera Routes
	// =========================

	// Admin: Camera management
	mux.Handle(
		"POST /api/sites/{siteID}/cameras",
		adminOnly(http.HandlerFunc(cameraHandler.Create)),
	)

	mux.Handle(
		"PUT /api/cameras/{cameraID}",
		adminOnly(http.HandlerFunc(cameraHandler.Update)),
	)

	mux.Handle(
		"DELETE /api/cameras/{cameraID}",
		adminOnly(http.HandlerFunc(cameraHandler.Delete)),
	)

	// User + Admin: Camera viewing
	mux.Handle(
		"GET /api/sites/{siteID}/cameras",
		protected(http.HandlerFunc(cameraHandler.GetAllBySite)),
	)

	mux.Handle(
		"GET /api/cameras/{cameraID}",
		protected(http.HandlerFunc(cameraHandler.GetByID)),
	)

	mux.Handle(
		"POST /api/cameras/{cameraID}/playback",
		protected(http.HandlerFunc(playbackHandler.Start)),
	)

	mux.Handle(
		"DELETE /api/playback/{leaseID}",
		protected(http.HandlerFunc(playbackHandler.Stop)),
	)

	mux.Handle(
		"POST /api/playback/{leaseID}/heartbeat",
		protected(http.HandlerFunc(playbackHandler.Heartbeat)),
	)
	// =========================
	// HLS File Serving
	// =========================

	mux.Handle(
		"GET /hls/streams/",
		http.StripPrefix(
			"/hls/streams/",
			http.FileServer(
				http.Dir(cfg.HLSDir),
			),
		),
	)

	// =========================
	// Shutdown
	// =========================

	shutdown := func() {
		hlsManager.Shutdown()
	}

	return mux, shutdown, nil
}
