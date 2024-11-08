package main

import (
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func main() {
	validate = validator.New()

	validate.RegisterValidation("filetype", validateFileType)
	validate.RegisterValidation("maxsize", validateFileSize)

	http.HandleFunc("/upload", uploadFileToS3)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}