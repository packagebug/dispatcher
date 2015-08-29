# Dispatcher of Packagebug

Dispatcher dispatchs a job for a bunch of worker. Dispatcher send the job as
a message to Amazon SQS, which is where the worker receive from. Each message
contain information about the package separated by commas. That information
is used by worker to fetch a bugs from the repository of package.

There are over 20K packages in the database, sending messsage to Amazon SQS
for each package require 20K requests. It's not optimal though. So, we send
message as a batch instead of 1 request per message.

## Setup
Make sure this enviroment variable already set

    DATABASE_URL

    PACKAGEBUG_SQS_ENDPOINT
    PACKAGEBUG_SQS_REGION

    AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY

Run

    go install
    packagebug-dispatcher
