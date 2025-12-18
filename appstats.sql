CREATE TABLE `users` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `user_id` VARCHAR(64) NOT NULL COMMENT '业务侧 user_id',
    `first_seen` DATE NOT NULL COMMENT '首次出现日期，用于新增用户',
    `platform` VARCHAR(32) NOT NULL COMMENT 'ios/android/web',
    `region` VARCHAR(64) DEFAULT NULL COMMENT '如 CN-Guangdong-Shenzhen',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_events` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `user_id` VARCHAR(64) NOT NULL,
    `event_type` VARCHAR(32) NOT NULL COMMENT 'login/heartbeat/action...',
    `platform` VARCHAR(32) NOT NULL,
    `region` VARCHAR(64) DEFAULT NULL,
    `event_time` DATETIME NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_user_events_time` (`event_time`),
    KEY `idx_user_events_user_time` (`user_id`, `event_time`)
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;