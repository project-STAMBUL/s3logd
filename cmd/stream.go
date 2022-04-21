package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nxadm/tail"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/context"
	"sigs.k8s.io/yaml"
)

var S3_REGION = os.Getenv("S3_REGION")
var S3_ENDPOINT = os.Getenv("S3_ENDPOINT")
var S3_BUCKET = os.Getenv("S3_BUCKET")
var S3_BUCKET_PATH = os.Getenv("S3_BUCKET_PATH")
var S3_ACCESS_KEY_ID = os.Getenv("S3_ACCESS_KEY_ID")
var S3_SECRET_ACCESS_KEY = os.Getenv("S3_SECRET_ACCESS_KEY")
var S3_SESSION_TOKEN = os.Getenv("S3_SESSION_TOKEN")

var s3session *session.Session

func init() {
	// it's an AWS tool, the S3_ prefixed environment variables is just to genericize things
	os.Setenv("AWS_ACCESS_KEY_ID", S3_ACCESS_KEY_ID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", S3_SECRET_ACCESS_KEY)
	os.Setenv("AWS_SESSION_TOKEN", S3_SESSION_TOKEN)

	Commands = append(Commands, StreamCmd)
}

var StreamCmd = &cli.Command{
	Name:    "stream",
	Aliases: []string{},
	Usage:   "reads log files and pushes them to an s3 bucket",
	Description: `reads log files and pushes them to an s3 bucket

		The following environment variables are available for S3 Object Storage.

		* S3_REGION            - region for endpoint
		* S3_ENDPOINT          - endpoint hostname
		* S3_BUCKET            - bucket for log uploads
		* S3_BUCKET_PATH       - base path in bucket to upload logs to
		* S3_ACCESS_KEY_ID     - key id for s3 access
		* S3_SECRET_ACCESS_KEY - secret key for s3 authentication
		* S3_SESSION_TOKEN     - optional s3 token
	`,
	ArgsUsage: "",
	Action:    StreamMain,

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "config",
			Value: "/config/streams.yaml",
			Usage: "absolute path to stash config",
		},
		&cli.StringFlag{
			Name:  "tempdir",
			Value: "/tmp",
			Usage: "temporary directory to hold logs in processing",
		},
	},
}

// this context is used to cancel all the streams at once
var streamContexts, cancelStreams = context.WithCancel(context.Background())

// this wait group is used to wait for streams to be done
var streamWg sync.WaitGroup

// gets set by --tempdir flag
var tempDir = ""

// the pod's hostname
var hostname = ""

func StreamMain(c *cli.Context) error {

	if h, err := os.Hostname(); err != nil {
		log.Fatalf("failed to get hostname: %s", err)
	} else {
		hostname = h
	}

	tempDir = c.String("tempdir")

	// ensure the tempdir exists
	os.MkdirAll(tempDir, os.ModePerm)

	// setup a session to be used for s3 activities.
	s3session = session.Must(session.NewSession(&aws.Config{
		Endpoint: &S3_ENDPOINT,
		Region:   &S3_BUCKET,
	}))

	// load in config
	configFile := c.String("config")
	conf, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("failed to load config '%s': %s", configFile, err)
	}

	// a list that will be set from the conf
	var files = make([]struct {
		File     string
		PushRate int
	}, 0)

	if err := yaml.Unmarshal(conf, &files); err != nil {
		log.Fatalf("failed to unmarshal config yaml '%s': %s", configFile, err)
	}

	// iterate over the files and start a stream for each one
	// it is possible to have no files, it was a concious decision to not do
	// anything special to call that out, perhaps someone just didn't want to
	// aggregate any log files
	for _, file := range files {
		go streamFileToS3(file.File, file.PushRate)
	}

	// hold process open indefinitely until killed, upon which it must cleanup
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Printf("kill signal caught, closing streams")
	cancelStreams()
	log.Printf("waiting for streams to complete")
	streamWg.Wait()

	return nil
}

func streamFileToS3(file string, pushRate int) {
	// ensure that the wait group gets closed and update the number of streams in the wait group
	defer streamWg.Done()
	streamWg.Add(1)

	// uploader for s3bucket
	uploader := s3manager.NewUploader(s3session)

	// the pulse starts low to give feedback to the user faster
	pulse := 30

	// start tailing the file and reopen rotated/trucated files
	stream, err := tail.TailFile(file, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		log.Fatalf("failed to open stream to file '%s': %s", file, err)
	}

	// the base name of the logfile
	// the temporary location to maintain the streamed copy
	// and where to put the file in the bucket
	filename, tempfile, destfile := setFilePaths(file)

	// filehandle for writing
	filehandle, err := os.OpenFile(tempfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("opening file '%s' failed: %s", tempfile, err)
	}

	// loop and select relevant task
	for {
		select {
		case line := <-stream.Lines: // add lines from the tail stream to the tempfile
			fmt.Fprintln(filehandle, line.Text)

		case <-streamContexts.Done(): // streams are cancelled, pulse will be set to 0 to run immediately
			pulse = 0

		case <-time.After(time.Duration(pulse) * time.Second):
			// ensure that the pulse gets updated to the pushRate
			pulse = pushRate

			log.Printf("%s - processing log chunk '%s'", file, tempfile)

			// need to know the filesize so that files with no content can be skipped
			info, err := filehandle.Stat()
			if err != nil {
				log.Fatalf("%s - statting file '%s' failed: %s", file, tempfile, err)
			}

			if info.Size() > 0 {
				log.Printf("%s - pushing log chunk '%s' to '%s' - %d", file, tempfile, destfile, info.Size())

				result, err := uploader.Upload(&s3manager.UploadInput{
					Bucket: aws.String(S3_BUCKET),
					Key:    aws.String(destfile),
					Body:   filehandle,
				})

				if err != nil {
					log.Printf("%s - failed to upload file '%s': %s", file, filename, err)
				}

				log.Printf("%s - file uploaded to, %s", file, aws.StringValue(&result.Location))

			} else {
				log.Printf("%s - skipping empty log chunk '%s'", file, tempfile)
			}

			// this is the end of needing this tempfile open, close in preparation to delete
			filehandle.Close()

			// if the context contains an error, it is likely due to cancelStreams() being called
			// this happens because of a signal to terminate the process
			if streamContexts.Err() != nil {
				log.Fatalf("%s - context is closed, closing stream", file)
				return
			}

			log.Printf("%s - deleting %s", file, tempfile)

			// cleanup tempfile that has been pushed to the bucket already
			if err := os.Remove(tempfile); err != nil {
				log.Fatalf("%s - deleting file '%s' failed: %s", file, tempfile, err)
			}

			log.Printf("%s - handling next log chunk", file)

			// use new filenames
			filename, tempfile, destfile = setFilePaths(file)

			// open a new filehandle for the new file
			filehandle, err = os.OpenFile(tempfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("%s - opening file '%s' failed: %s", file, tempfile, err)
			}
		}
	}
}

// convenience function for knowing the base name of a logfile
// what temporary file path to use
// and where to upload the file in a bucked
func setFilePaths(file string) (string, string, string) {
	filename := filepath.Base(file)
	filename = filename + "." + fmt.Sprintf("%d", time.Now().Unix())
	tempfile := filepath.Join(tempDir, filename)
	destfile := filepath.Join(S3_BUCKET_PATH, time.Now().Format("2006/01/02"), hostname, filename)
	return filename, tempfile, destfile
}
