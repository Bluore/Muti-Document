USE mutidoc;


CREATE TABLE `user`
(
    ID VARCHAR(50) not null,
    NickName VARCHAR(100) NOT NULL ,
    Email VARCHAR(100) NOT NULL ,
    Avatar VARCHAR(100) NOT NULL DEFAULT 'default-avatar',
    CreateAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP(),
    DeleteAt DATETIME NULL DEFAULT NULL,

    PRIMARY KEY (ID),
    UNIQUE KEY (NickName),
    UNIQUE KEY (Email)
);
