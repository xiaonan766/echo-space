-- Execute this migration manually before enabling /web/ucenter/postVideo.

ALTER TABLE video_info_post
    MODIFY COLUMN video_cover VARCHAR(255) NOT NULL COMMENT '视频封面',
    MODIFY COLUMN category_id INT NULL COMMENT '分类ID';

ALTER TABLE video_info
    MODIFY COLUMN video_cover VARCHAR(255) NOT NULL COMMENT '视频封面';

CREATE TABLE IF NOT EXISTS video_transcode_message (
    message_id VARCHAR(32) NOT NULL COMMENT '消息ID',
    file_id VARCHAR(20) NOT NULL COMMENT '投稿文件ID',
    video_id VARCHAR(10) NOT NULL COMMENT '视频ID',
    user_id VARCHAR(10) NOT NULL COMMENT '用户ID',
    upload_id VARCHAR(15) NOT NULL COMMENT '上传任务ID',
    message_status TINYINT NOT NULL DEFAULT 0 COMMENT '0待发布 1已发布 2处理中 3成功 4等待重试 5死亡',
    payload JSON NOT NULL COMMENT '转码任务快照',
    retry_count INT NOT NULL DEFAULT 0 COMMENT '转码失败次数',
    next_retry_time DATETIME NULL COMMENT '下次发布时间',
    locked_until DATETIME NULL COMMENT '处理租约截止时间',
    lock_token VARCHAR(32) NOT NULL DEFAULT '' COMMENT '本次处理租约令牌',
    last_error VARCHAR(500) NOT NULL DEFAULT '' COMMENT '最后错误',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id),
    UNIQUE KEY uk_video_transcode_file_id (file_id),
    UNIQUE KEY uk_video_transcode_upload (user_id, upload_id),
    KEY idx_video_transcode_retry (message_status, next_retry_time),
    KEY idx_video_transcode_lease (message_status, locked_until),
    KEY idx_video_transcode_video (video_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频异步转码本地消息表';
