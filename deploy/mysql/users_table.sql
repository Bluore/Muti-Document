USE mutidoc;

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users`
(
    ID VARCHAR(50) not null,
    NickName VARCHAR(100) NOT NULL ,
    Email VARCHAR(100) NOT NULL ,
    Avatar VARCHAR(100) NOT NULL DEFAULT 'default-avatar.png',
    CreateAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP(),
    UpdateAt DATETIME NOT NULL DEFAULT (CreateAt),
    DeleteAt DATETIME NULL DEFAULT NULL,

    PRIMARY KEY (ID),
    UNIQUE (NickName),
    UNIQUE (Email),
    INDEX (DeleteAt)
);
