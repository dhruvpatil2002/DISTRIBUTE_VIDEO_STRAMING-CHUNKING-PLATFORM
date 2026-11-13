CREATE TABLE cameras (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    site_id UUID NOT NULL,

    name VARCHAR(150) NOT NULL,

    rtsp_url TEXT NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'registered',

    hls_url TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cameras_site
        FOREIGN KEY (site_id)
        REFERENCES sites(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_camera_status
        CHECK (
            status IN (
                'registered',
                'connecting',
                'connected',
                'disconnected',
                'failed'
            )
        )
);

CREATE INDEX idx_cameras_site_id
ON cameras(site_id);

CREATE INDEX idx_cameras_status
ON cameras(status);