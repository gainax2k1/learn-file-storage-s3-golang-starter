package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20 // Bit shift 10 to the left 20 times to get 10Mb

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't retrieve form file data and/or header", err)
		return
	}

	mediaHeader := fileHeaders.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(mediaHeader)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse mime", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "filetype not jpeg or png", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Authenticated user is not the video owner", err)
		return
	}

	// str := base64.StdEncoding.EncodeToString(imageData)

	//root := cfg.assetsRoot // temp for figuring out
	mediaSuffix := (strings.Split(mediaType, "/"))[1]

	key := make([]byte, 32)
	rand.Read(key)
	keyString := base64.RawURLEncoding.EncodeToString(key)
	// ****** change videoidstring here to new generated string ***********
	joinedPath := filepath.Join(cfg.assetsRoot, (keyString + "." + mediaSuffix))
	newFile, err := os.Create(joinedPath)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to create file", err)
		return
	}
	defer newFile.Close() // alwasy defer a close on create

	_, err = io.Copy(newFile, fileData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to copy file", err)
		return
	}

	thumbnailURL := "http://localhost:8091/" + joinedPath //"data:" + mediaType + ";base64," + str
	videoMetadata.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
