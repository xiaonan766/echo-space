ALTER TABLE video_info_post
  ADD COLUMN content_type TINYINT NOT NULL DEFAULT 0 COMMENT '稿件类型：0=视频，1=图片'
  AFTER category_id;

CREATE INDEX idx_video_info_post_content_type
  ON video_info_post (content_type, last_update_time);
