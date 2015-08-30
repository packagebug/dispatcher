package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sqs"
	_ "github.com/lib/pq"
)

const (
	MAX_BATCH_SIZE = 250 * 1024 // 250KB
)

var (
	PACKAGEBUG_DB           = os.Getenv("DATABASE_URL")
	PACKAGEBUG_SQS_ENDPOINT = os.Getenv("PACKAGEBUG_SQS_ENDPOINT")
	PACKAGEBUG_SQS_REGION   = os.Getenv("PACKAGEBUG_SQS_REGION")
)

var (
	dbconn  *sql.DB
	sqsconn *sqs.SQS
)

type Package struct {
	Id    string
	Host  string
	Owner string
	Repo  string
}

// sendMessages sends a batchs of message to Amazon SQS. Each batch must not
// exceed 250KB.
func sendMessages(packages []Package) error {
	batchSize := 0
	batch := new(sqs.SendMessageBatchInput)
	batch.QueueUrl = aws.String(PACKAGEBUG_SQS_ENDPOINT)

	for _, p := range packages {
		// compose a batch
		msgBody := fmt.Sprintf("%s,%s,%s,%s", p.Id, p.Host, p.Owner, p.Repo)
		batchSize += len(msgBody)
		m := &sqs.SendMessageBatchRequestEntry{
			Id:          aws.String(p.Id),
			MessageBody: aws.String(msgBody),
		}
		batch.Entries = append(batch.Entries, m)

		// if batch contain 10 messages and total size is < 250KB then submit
		// the batch
		if len(batch.Entries) == 10 && batchSize < MAX_BATCH_SIZE {
			// transmit the batch
			_, err := sqsconn.SendMessageBatch(batch)
			if err != nil {
				return err
			}
			log.Printf("[dispatcher] %d bytes messages transmitted.\n", batchSize)
			// restore the state
			batchSize = 0
			batch = new(sqs.SendMessageBatchInput)
			batch.QueueUrl = aws.String(PACKAGEBUG_SQS_ENDPOINT)
		}
	}

	return nil
}

// dispatchJobs dispatchs a jobs via Amazon SQS.
func dispatchJobs() {
	// get data of all packages from the database
	rows, err := dbconn.Query(`
	SELECT package_id, package_host, package_owner, package_repo
	FROM packages
	ORDER BY package_id ASC;`)
	if err != nil {
		log.Printf("[dispatcher] error sql query: %v\n", err)
		return
	}
	defer rows.Close()

	var packages []Package
	for rows.Next() {
		var p Package
		err := rows.Scan(&p.Id, &p.Host, &p.Owner, &p.Repo)
		if err != nil {
			log.Printf("[dispatcher] iterate rows: %v\n", err)
			return
		}
		packages = append(packages, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[dispatcher] iterate rows: %v\n", err)
		return
	}

	// send message to Amazon SQS as a batch
	err = sendMessages(packages)
	if err != nil {
		log.Printf("[dispatcher] error send messages: %v\n", err)
		return
	}
}

func main() {
	var err error

	// connect to the database
	dbconn, err = sql.Open("postgres", PACKAGEBUG_DB)
	if err != nil {
		log.Fatal(err)
	}

	// make sure the database up
	err = dbconn.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// set up aws SDK credentials & config
	cred := credentials.NewEnvCredentials()
	_, err = cred.Get()
	if err != nil {
		log.Fatal(err)
	}
	config := aws.NewConfig()
	config.Credentials = cred
	config.Endpoint = &PACKAGEBUG_SQS_ENDPOINT
	config.Region = &PACKAGEBUG_SQS_REGION

	sqsconn = sqs.New(config)
	// dispatch jobs once a day
	for {
		<-time.After(24 * time.Hour)
		go dispatchJobs()
	}
}
