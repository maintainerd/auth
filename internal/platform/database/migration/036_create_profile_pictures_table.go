package migration

import (
	"gorm.io/gorm"
)

// CreateProfilePicturesTable stores uploaded avatar bytes in a table of their
// own, separate from profiles.
//
// A SIDE TABLE, not a column on profiles, and that is the whole point of this
// migration. An avatar is up to 2 MiB; profiles is read on every profile
// fetch, every profile list, and every preload of a user's Profile relation.
// A bytea column there means a `SELECT *` — which is what an ORM emits by
// default — drags the image into memory on all of those paths. At a million
// users that is the difference between a profile list costing kilobytes and
// costing gigabytes.
//
// Kept out of profiles, the bytes are read by exactly one endpoint, and no
// existing query changes or has to remember to exclude a column.
//
// This is also a TEMPORARY home. profiles.profile_url is the durable contract:
// an uploaded image sets it to this service's own serve path, so a profile
// always simply has a URL. When object storage arrives, uploads write there
// and set profile_url to that URL instead — the profile model, the API
// response and every client stay exactly as they are, and this table is
// drained and dropped.
func CreateProfilePicturesTable(db *gorm.DB) error {
	sql := `
-- CREATE TABLE
CREATE TABLE IF NOT EXISTS profile_pictures (
    profile_picture_id   BIGSERIAL PRIMARY KEY,

    -- One picture per profile, enforced by the UNIQUE below. Replacing an
    -- avatar overwrites this row rather than accumulating versions: nothing
    -- reads history, and keeping it would grow unboundedly with every edit.
    profile_id           BIGINT NOT NULL,

    -- The image bytes. Postgres TOASTs a value this size out of the main heap
    -- automatically, so even a scan of this table does not pay for them unless
    -- the column is selected.
    data                 BYTEA NOT NULL,

    -- The media type established by DECODING the upload, never the
    -- Content-Type the client sent. It is echoed back when serving, so a
    -- client-supplied value would let an uploader choose how a browser
    -- interprets their bytes.
    content_type         VARCHAR(64) NOT NULL,

    -- Strong validator for the serve endpoint, so a repeat view is a 304 and
    -- never re-reads the bytes. Without it every avatar render is a full read
    -- of up to 2 MiB.
    etag                 VARCHAR(64) NOT NULL,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- 2 MiB. Enforced here as well as in the handler so a writer that does not
    -- come through the API cannot use a profile as general-purpose storage.
    CONSTRAINT chk_profile_pictures_size CHECK (octet_length(data) <= 2097152),
    -- Only the raster types the uploader decodes and re-verifies. SVG is absent
    -- deliberately: it is a document that can carry script, and serving one
    -- from this origin would be stored XSS against every viewer of that avatar.
    CONSTRAINT chk_profile_pictures_type CHECK (content_type IN ('image/png', 'image/jpeg', 'image/webp', 'image/gif'))
);

-- ADD CONSTRAINTS (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_profile_pictures_profile_id'
    ) THEN
        -- CASCADE is what keeps an erasure request honest: deleting the profile
        -- deletes the image, with no cleanup job to forget to run and no
        -- orphaned bytes left behind.
        ALTER TABLE profile_pictures
            ADD CONSTRAINT fk_profile_pictures_profile_id FOREIGN KEY (profile_id)
            REFERENCES profiles(profile_id) ON DELETE CASCADE;
    END IF;
END $$;

-- ADD INDEXES
--
-- UNIQUE rather than a plain index: it is both the lookup path for the serve
-- endpoint and the guarantee that a profile cannot accumulate avatars.
CREATE UNIQUE INDEX IF NOT EXISTS uq_profile_pictures_profile_id
    ON profile_pictures (profile_id);
`
	return db.Exec(sql).Error
}
