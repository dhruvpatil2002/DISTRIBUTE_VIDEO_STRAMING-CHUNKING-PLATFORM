package routes

import (
    "context"
    "fmt"
    "net/http"
    "log"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/config"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/handler"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/repository"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/service"
    "github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/stream"
)

func New(
    db *pgxpool.Pool,
    cfg *config.Config,
) (http.Handler, func(), error) {

    mux := http.NewServeMux()

    // Repositories
    siteRepo := repository.NewSiteRepository(db)
    cameraRepo := repository.NewCameraRepository(db)

    // Stream managers
    rtspManager := stream.NewMediaMTXManager(
        cfg.MediaMTXAPI,
        cfg.MediaMTXRTSP,
    )

   hlsManager := stream.NewHLSManager(
    cfg.FFmpegPath,
    cfg.HLSDir,
    cfg.StreamGracePeriod,
    time.Minute,
    2*time.Second,
    5,
)

    // Restore registered cameras.
    ctx := context.Background()

    cameras, err := cameraRepo.GetAllForStreaming(ctx)
    if err != nil {
        hlsManager.Shutdown()
        return nil, nil, fmt.Errorf(
            "load registered cameras: %w", err,
        )
    }

    for _, camera := range cameras {
        if err := rtspManager.RegisterCamera(
            ctx,
            camera.ID,
            camera.RTSPURL,
        ); err != nil {
            hlsManager.Shutdown()
            return nil, nil, fmt.Errorf(
                "restore camera %s: %w",
                camera.ID,
                err,
            )
        }

        log.Printf(
            "Registered RTSP source for camera %s",
            camera.ID,
        )
    }

    // Services
    siteService := service.NewSiteService(siteRepo)

    cameraService := service.NewCameraService(
        cameraRepo,
        rtspManager,
        hlsManager,
    )

    // Handlers
    siteHandler := handlers.NewSiteHandler(siteService)
    cameraHandler := handlers.NewCameraHandler(cameraService)
    playbackHandler := handlers.NewPlaybackHandler(cameraService)

    // Site routes
    mux.HandleFunc("POST /api/sites", siteHandler.Create)
    mux.HandleFunc("GET /api/sites", siteHandler.GetAll)
    mux.HandleFunc("GET /api/sites/{siteID}", siteHandler.GetByID)
    mux.HandleFunc("PUT /api/sites/{siteID}", siteHandler.Update)
    mux.HandleFunc("DELETE /api/sites/{siteID}", siteHandler.Delete)

    // Camera routes
    mux.HandleFunc(
        "POST /api/sites/{siteID}/cameras",
        cameraHandler.Create,
    )

    mux.HandleFunc(
        "GET /api/sites/{siteID}/cameras",
        cameraHandler.GetAllBySite,
    )

    mux.HandleFunc(
        "GET /api/cameras/{cameraID}",
        cameraHandler.GetByID,
    )

    mux.HandleFunc(
        "PUT /api/cameras/{cameraID}",
        cameraHandler.Update,
    )

    mux.HandleFunc(
        "DELETE /api/cameras/{cameraID}",
        cameraHandler.Delete,
    )

    // Playback routes
    mux.HandleFunc(
        "POST /api/cameras/{cameraID}/playback",
        playbackHandler.Start,
    )

    mux.HandleFunc(
        "DELETE /api/playback/{leaseID}",
        playbackHandler.Stop,
    )
	mux.HandleFunc(
    "POST /api/playback/{leaseID}/heartbeat",
    playbackHandler.Heartbeat,
)

    // HLS file serving
    mux.Handle(
        "GET /hls/streams/",
        http.StripPrefix(
            "/hls/streams/",
            http.FileServer(
                http.Dir(cfg.HLSDir),
            ),
        ),
    )

    shutdown := func() {
        hlsManager.Shutdown()
    }

    return mux, shutdown, nil
}