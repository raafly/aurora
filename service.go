package main

import (
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

const (
	maxUploadSize = 5 << 20 
)

var (
	S3_BUCKET   string
	SECRET_KEY  string
	REGION      string
	BUCKET_NAME string
	validate    *validator.Validate
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	S3_BUCKET = os.Getenv("AWS_ACCESS_KEY")
	SECRET_KEY = os.Getenv("AWS_SECRET_KEY")
	REGION = os.Getenv("AWS_REGION")
	BUCKET_NAME = os.Getenv("AWS_BUCKET_NAME")

	validate = validator.New()
	validate.RegisterValidation("filetype", validateFileType)
	validate.RegisterValidation("maxsize", validateFileSize)
}

type Request struct {
	Image *multipart.FileHeader `form:"image" validate:"required,filetype=jpg|png,maxsize=5MB"`
	HW    string                `form:"hw"`
}

func uploadFileToS3(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	sess, err := createAWSSession()
	if err != nil {
		http.Error(w, "Failed to create AWS session", http.StatusInternalServerError)
		return
	}

	tmpFile, err := createTempFileFromMultipart(file)
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()

	err = uploadToS3(sess, tmpFile, "uploaded-file.png")
	if err != nil {
		http.Error(w, "Failed to upload file to S3", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"success": true,
		"message": "success upload to s3",
	}
	json.NewEncoder(w).Encode(response)
}

func createAWSSession() (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region:      aws.String(REGION),
		Credentials: credentials.NewStaticCredentials(S3_BUCKET, SECRET_KEY, ""),
	})
}

func uploadToS3(sess *session.Session, file *os.File, fileName string) error {
	svc := s3.New(sess)
	_, err := svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(BUCKET_NAME),
		Key:         aws.String(fileName),
		Body:        file,
		ContentType: aws.String("application/octet-stream"),
	})
	return err
}

func createTempFileFromMultipart(file multipart.File) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		tmpFile.Close()
		return nil, err
	}

	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		tmpFile.Close()
		return nil, err
	}

	return tmpFile, nil
}

func validateFileType(fl validator.FieldLevel) bool {
	file, ok := fl.Field().Interface().(*multipart.FileHeader)
	if !ok {
		return false
	}

	filename := file.Filename
	return filename[len(filename)-4:] == ".jpg" || filename[len(filename)-4:] == ".png"
}

func validateFileSize(fl validator.FieldLevel) bool {
	file, ok := fl.Field().Interface().(*multipart.FileHeader)
	if !ok {
		return false
	}

	const maxSize = 5 * 1024 * 1024
	return file.Size <= maxSize
}
