USE mutidoc;

DROP TABLE IF EXISTS `agent_sessions`;
CREATE TABLE agent_sessions
(
    `id` VARCHAR(50) NOT NULL,
    `user_id` VARCHAR(50) NOT NULL,
    `title` VARCHAR(128) NOT NULL DEFAULT '',
    `history` TEXT NOT NULL COMMENT '用户界面可见的对话与思考内容(JSONL)',
    `memory` TEXT NOT NULL COMMENT '大模型实际输入的消息(JSONL)',
    `status` TINYINT NOT NULL DEFAULT 0,

    `version` INTEGER NOT NULL DEFAULT 1,
    `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state` TINYINT NOT NULL DEFAULT 0,

    PRIMARY KEY (`id`),
    INDEX (`del_state`),
    INDEX (`user_id`, `create_at`, `del_state`)
);
