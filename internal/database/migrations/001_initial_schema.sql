
-- =========================================
-- USERS TABLE
-- =========================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================
-- VIDEOS TABLE
-- =========================================

CREATE TABLE IF NOT EXISTS videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    title VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,

    original_url TEXT,
    master_playlist_url TEXT,

    mime_type VARCHAR(100),
    file_size_bytes BIGINT NOT NULL DEFAULT 0,

    duration_seconds DOUBLE PRECISION,
    width INTEGER,
    height INTEGER,

    status VARCHAR(30) NOT NULL DEFAULT 'uploaded'
        CHECK (status IN (
            'uploaded',
            'queued',
            'processing',
            'completed',
            'failed'
        )),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id
ON videos(user_id);

CREATE INDEX IF NOT EXISTS idx_videos_status
ON videos(status);

-- =========================================
-- TRANSCODING JOBS TABLE
-- =========================================

CREATE TABLE IF NOT EXISTS transcoding_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,

    job_type VARCHAR(50) NOT NULL DEFAULT 'transcode',

    status VARCHAR(30) NOT NULL DEFAULT 'queued'
        CHECK (status IN (
            'queued',
            'processing',
            'completed',
            'failed',
            'cancelled'
        )),

    progress INTEGER NOT NULL DEFAULT 0
        CHECK (progress BETWEEN 0 AND 100),

    attempts INTEGER NOT NULL DEFAULT 0,

    worker_id VARCHAR(255),
    error_message TEXT,

    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_video_id
ON transcoding_jobs(video_id);

CREATE INDEX IF NOT EXISTS idx_jobs_status
ON transcoding_jobs(status);

-- =========================================
-- VIDEO VARIANTS TABLE
-- =========================================

CREATE TABLE IF NOT EXISTS video_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,

    quality VARCHAR(20) NOT NULL,

    width INTEGER NOT NULL,
    height INTEGER NOT NULL,

    bitrate INTEGER,

    playlist_url TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(video_id, quality)
);

CREATE INDEX IF NOT EXISTS idx_variants_video_id
ON video_variants(video_id);
