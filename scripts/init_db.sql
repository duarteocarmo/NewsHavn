CREATE TABLE IF NOT EXISTS articles (
    ID VARCHAR(255) PRIMARY KEY,
    Title TEXT NOT NULL,
    Link TEXT NOT NULL UNIQUE,
    Date DATETIME,
    Content TEXT NOT NULL,
    Source VARCHAR(255),
    Category VARCHAR(255),
    TranslatedContent TEXT,
    TranslatedTitle TEXT
);
