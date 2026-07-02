-- Dynamic feed mixed content migration.
-- Execute this migration manually before enabling image posts in follower dynamic feed.

ALTER TABLE user_dynamic_feed
    ADD COLUMN event_type TINYINT NOT NULL DEFAULT 1 COMMENT 'event type 1: video, 2: image' AFTER video_id;

ALTER TABLE user_dynamic_feed
    DROP INDEX uk_user_dynamic_video,
    ADD UNIQUE KEY uk_user_dynamic_content (user_id, event_type, video_id),
    DROP INDEX idx_user_dynamic_time,
    ADD KEY idx_user_dynamic_time (user_id, dynamic_time, event_type, video_id),
    DROP INDEX idx_user_author_dynamic_time,
    ADD KEY idx_user_author_dynamic_time (user_id, author_user_id, dynamic_time, event_type, video_id);
