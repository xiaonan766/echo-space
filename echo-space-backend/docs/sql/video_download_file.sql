-- 视频下载版文件状态，执行一次即可。

ALTER TABLE video_info_file
    ADD COLUMN download_status TINYINT NOT NULL DEFAULT 0
    COMMENT '下载版状态 0：未生成 1：生成中 2：成功 3：失败'
    AFTER duration,
    ADD COLUMN download_file_path VARCHAR(255) NULL
    COMMENT '下载版视频路径'
    AFTER download_status;
