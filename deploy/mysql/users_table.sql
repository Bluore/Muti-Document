USE mutidoc;

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users`
(
    `id` VARCHAR(50) not null,
    `nick_name` VARCHAR(100) NOT NULL ,
    `email` VARCHAR(100) NOT NULL ,
    `avatar` VARCHAR(100) NOT NULL DEFAULT 'default-avatar.png',
    `password` VARCHAR(255) NOT NULL ,

    `version` INTEGER NOT NULL DEFAULT 1,
    `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state` TINYINT NOT NULL DEFAULT 0,

    PRIMARY KEY (`id`),
    UNIQUE (`nick_name`),
    UNIQUE (`email`),
    INDEX (`del_state`)
);
