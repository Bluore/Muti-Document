USE mutidoc;

DROP TABLE IF EXISTS `documents`;
CREATE TABLE documents
(
    `id` VARCHAR(50) NOT NULL,
    `owner_id` VARCHAR(50) NOT NULL,
    `summary` VARCHAR(255) NOT NULL DEFAULT 'new document',
    `content` TEXT NOT NULL,

    `version` INTEGER NOT NULL DEFAULT 1,
    `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state` TINYINT NOT NULL DEFAULT 0,

    PRIMARY KEY (`id`),
    INDEX (`del_state`),
    INDEX (`owner_id`, `create_at`, `del_state`)
);

DROP TABLE IF EXISTS `grant_documents_permission`;
CREATE TABLE `grant_documents_permission`
(
    `permission_id` VARCHAR(50) NOT NULL ,
    `user_id` VARCHAR(50) NOT NULL ,
    `document_id` VARCHAR(50) NOT NULL ,

    `version` INTEGER NOT NULL DEFAULT 1,
    `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state` TINYINT NOT NULL DEFAULT 0,

    PRIMARY KEY (`permission_id`),
    INDEX (`user_id`, `document_id`)
)