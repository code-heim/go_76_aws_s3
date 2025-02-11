package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	bucketName string
	s3Region   string
	uploadDir  string
	s3Client   *s3.Client
)

const templateFile = "templates/index.html"

func main() {
	// Read command-line arguments
	flag.StringVar(&bucketName, "bucket", "", "S3 bucket name (required)")
	flag.StringVar(&s3Region, "region", "", "AWS region (required)")
	flag.StringVar(&uploadDir, "uploadDir", "uploads/", "S3 directory to store images")
	flag.Parse()

	// Ensure required flags are provided
	if bucketName == "" || s3Region == "" {
		fmt.Println("Usage: go run main.go -bucket=<S3_BUCKET> -region=<AWS_REGION> -uploadDir=<S3_DIRECTORY>")
		os.Exit(1)
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3Region))
	if err != nil {
		panic(fmt.Sprintf("Failed to load AWS config: %v", err))
	}

	s3Client = s3.NewFromConfig(cfg)

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/upload", uploadHandler)

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(uploadDir),
	})
	if err != nil {
		http.Error(w, "Unable to read images", http.StatusInternalServerError)
		return
	}

	var images []string
	for _, item := range resp.Contents {
		presignClient := s3.NewPresignClient(s3Client)
		presignedURL, err := presignClient.PresignGetObject(context.TODO(),
			&s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(*item.Key),
			},
		)
		if err != nil {
			continue
		}
		images = append(images, presignedURL.URL)
	}

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, images)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Construct file path in S3
	s3Key := filepath.Join(uploadDir, header.Filename)

	// Upload to S3
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to upload to S3", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
