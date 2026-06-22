-- 视频投稿与正式视频都需要保留下载权限，执行一次即可。

ALTER TABLE video_info_post
    ADD COLUMN download_permission TINYINT(1) NOT NULL DEFAULT 1
    COMMENT '下载权限 0：禁止下载 1：允许下载'
    AFTER interaction;

ALTER TABLE video_info
    ADD COLUMN download_permission TINYINT(1) NOT NULL DEFAULT 1
    COMMENT '下载权限 0：禁止下载 1：允许下载'
    AFTER interaction;
