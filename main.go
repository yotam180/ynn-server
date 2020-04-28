package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const (
	uploadPath = "./uploads"
	metaPath   = "./metadata" // For passwords storage, etc
)

func checkAccess(namespace, file string, password []byte) bool {
	fPath := path.Join(metaPath, namespace, file)
	if _, err := os.Stat(fPath); os.IsNotExist(err) {
		return true // No password -> We can access the file
	}

	pass, err := ioutil.ReadFile(fPath)
	if err != nil {
		fmt.Println(err.Error())
		return false // Unknown error -> no access for you :(
	}

	return bytes.Equal(pass, password)
}

func setPassword(namespace, file string, password []byte) error {
	fPath := path.Join(metaPath, namespace, file)

	if err := os.MkdirAll(path.Dir(fPath), os.ModePerm); err != nil {
		return err
	}

	if err := ioutil.WriteFile(fPath, password, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func getFile(c *gin.Context) {
	fPath := path.Join(uploadPath, c.Param("namespace"), c.Param("filePath"))

	pass := c.Request.Header.Get("Authorization")
	if !checkAccess(c.Param("namespace"), c.Param("filePath"), []byte(pass)) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Bad password",
		})
		return
	}

	if _, err := os.Stat(fPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File does not exist",
		})
		return
	}

	c.File(fPath)
}

func uploadFile(c *gin.Context) {
	fileName := filepath.Base(c.Param("filePath"))
	fPath := path.Join(uploadPath, c.Param("namespace"), fileName)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	pass := c.Request.Header.Get("Authorization") // Contains raw password
	if pass != "" {
		err = setPassword(c.Param("namespace"), fileName, []byte(pass))
	}

	fmt.Println("Creating file ", fPath)
	err = os.MkdirAll(filepath.Dir(fPath), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.SaveUploadedFile(file, fPath)
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"file_name": fileName,
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	port = ":" + port

	r := gin.Default()
	r.POST("/files/:namespace/:filePath", uploadFile)
	r.GET("/files/:namespace/:filePath", getFile)
	r.Run(port)
}
