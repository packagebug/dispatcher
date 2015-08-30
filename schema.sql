CREATE TABLE packages (
    package_id SERIAL PRIMARY KEY,
    package_path text UNIQUE NOT NULL,
    package_host text NOT NULL,
    package_owner text NOT NULL,
    package_repo text NOT NULL,
    package_etag text
);

CREATE TABLE bugs (
    bug_package_id integer REFERENCES packages(package_id) ON DELETE CASCADE,
    bug_id SERIAL PRIMARY KEY,
    bug_number integer NOT NULL,
    bug_url text NOT NULL,
    bug_title text NOT NULL
);


