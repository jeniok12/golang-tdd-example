-- ..._create_recipient_table.up
CREATE TABLE recipients
(
    id SERIAL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    CONSTRAINT recipients_pkey PRIMARY KEY (id)
);