-- Dynamic feed hybrid fanout migration.
-- Execute this migration manually before enabling the mixed fanout dynamic feed.

CREATE TABLE IF NOT EXISTS dynamic_event (
    event_id VARCHAR(32) NOT NULL COMMENT 'event id',
    video_id VARCHAR(10) NOT NULL COMMENT 'video id',
    author_user_id VARCHAR(10) NOT NULL COMMENT 'author user id',
    dynamic_time DATETIME NOT NULL COMMENT 'dynamic display time',
    event_type TINYINT NOT NULL DEFAULT 1 COMMENT 'event type 1: video published',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (event_id),
    UNIQUE KEY uk_dynamic_event_video (video_id),
    KEY idx_dynamic_event_author_time (author_user_id, dynamic_time, video_id),
    KEY idx_dynamic_event_time (dynamic_time, video_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='dynamic event source';

CREATE TABLE IF NOT EXISTS user_dynamic_feed (
    feed_id VARCHAR(32) NOT NULL COMMENT 'feed id',
    user_id VARCHAR(10) NOT NULL COMMENT 'receiver user id',
    author_user_id VARCHAR(10) NOT NULL COMMENT 'author user id',
    video_id VARCHAR(10) NOT NULL COMMENT 'video id',
    dynamic_time DATETIME NOT NULL COMMENT 'dynamic display time',
    push_time DATETIME NOT NULL COMMENT 'fanout time',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (feed_id),
    UNIQUE KEY uk_user_dynamic_video (user_id, video_id),
    KEY idx_user_dynamic_time (user_id, dynamic_time, video_id),
    KEY idx_user_author_dynamic_time (user_id, author_user_id, dynamic_time, video_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='user dynamic feed inbox';

CREATE TABLE IF NOT EXISTS dynamic_feed_message (
    message_id VARCHAR(32) NOT NULL COMMENT 'message id',
    event_id VARCHAR(32) NOT NULL COMMENT 'event id',
    video_id VARCHAR(10) NOT NULL COMMENT 'video id',
    author_user_id VARCHAR(10) NOT NULL COMMENT 'author user id',
    message_status TINYINT NOT NULL DEFAULT 0 COMMENT '0 wait publish 1 published 2 processing 3 success 4 retry wait 5 dead',
    payload JSON NOT NULL COMMENT 'message payload',
    retry_count INT NOT NULL DEFAULT 0 COMMENT 'retry count',
    next_retry_time DATETIME NULL COMMENT 'next retry time',
    locked_until DATETIME NULL COMMENT 'processing lease end time',
    lock_token VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'processing lock token',
    last_error VARCHAR(500) NOT NULL DEFAULT '' COMMENT 'last error',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id),
    KEY idx_dynamic_feed_message_retry (message_status, next_retry_time),
    KEY idx_dynamic_feed_message_lease (message_status, locked_until),
    KEY idx_dynamic_feed_message_event (event_id),
    KEY idx_dynamic_feed_message_video (video_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='dynamic feed outbox message';

INSERT INTO dynamic_event (
    event_id, video_id, author_user_id, dynamic_time, event_type, create_time, update_time
)
SELECT
    CONCAT('video_', vi.video_id),
    vi.video_id,
    vi.user_id,
    vi.last_update_time,
    1,
    NOW(),
    NOW()
FROM video_info vi
ON DUPLICATE KEY UPDATE
    author_user_id = VALUES(author_user_id),
    dynamic_time = VALUES(dynamic_time),
    event_type = VALUES(event_type),
    update_time = NOW();

INSERT INTO user_dynamic_feed (
    feed_id, user_id, author_user_id, video_id, dynamic_time, push_time, create_time, update_time
)
SELECT
    CONCAT(uf.user_id, '_', vi.video_id),
    uf.user_id,
    vi.user_id,
    vi.video_id,
    vi.last_update_time,
    NOW(),
    NOW(),
    NOW()
FROM user_focus uf
INNER JOIN video_info vi ON vi.user_id = uf.focus_user_id
WHERE uf.focus_time <= vi.last_update_time
ON DUPLICATE KEY UPDATE
    author_user_id = VALUES(author_user_id),
    dynamic_time = VALUES(dynamic_time),
    update_time = NOW();
