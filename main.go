package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

func getPageAsText(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Go-User-Agent")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func getFinalURLAfterRedirects(url string) (string, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Go-User-Agent")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	return resp.Request.URL.String(), nil
}

func getPostJson(id string) (string, error) {
	newUrl, err := getFinalURLAfterRedirects(fmt.Sprintf("https://v.redd.it/%s", id))
	if err != nil {
		return "", err
	}
	pageText, err := getPageAsText(fmt.Sprintf(`%s.json`, newUrl))
	if err != nil {
		return "", err
	}

	return pageText, nil
}

func getVideoUrl(json string) (string, error) {
	pattern := regexp.MustCompile(`v\.redd\.it/.*?/DASH_.*?.mp4`)
	match := "https://" + pattern.FindString(json)
	if match == "" {
		return "", errors.New("no video url found")
	}

	return match, nil
}
func convertVideoToAudioURL(url string) string {
	pattern := regexp.MustCompile(`DASH_.*?.mp4`)
	return pattern.ReplaceAllString(url, "DASH_audio.mp4")
}
func isURLOkay(url string) bool {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Go-User-Agent")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	return resp.StatusCode == 200
}
func main() {
	router := gin.Default()

	router.GET("/:id", func(ctx *gin.Context) {
		stripEndings := regexp.MustCompile(`(\.mp4)|(\.mkv)`)

		id := ctx.Params.ByName("id")
		id = stripEndings.ReplaceAllString(id, "")

		if !isURLOkay("https://v.redd.it/" + id) {
			ctx.String(400, "Invalid url given")
			return
		}

		json, err := getPostJson(id)
		if err != nil {
			ctx.String(400, "Error getting metadata: %s", err)
			return
		}

		videoURL, _ := getVideoUrl(json)

		audioURL := convertVideoToAudioURL(videoURL)
		hasAudio := isURLOkay(audioURL)

		var command exec.Cmd
		if hasAudio {
			command = *exec.Command("ffmpeg", "-i", videoURL, "-i", audioURL, "-vf", "scale='min(iw,720)':-2", "-f", "matroska", "pipe:1")
		} else {
			command = *exec.Command("ffmpeg", "-i", videoURL, "-vf", "scale='min(iw,720)':-2", "-f", "matroska", "pipe:1")
		}

		ctx.Writer.Header().Set("Content-type", "video/mkv")
		ctx.Writer.Header().Set("Content-Disposition", "attachment;filename=video.mkv")
		ctx.Stream(func(w io.Writer) bool {
			pr, pw := io.Pipe()
			command.Stdout = pw
			go io.Copy(w, pr)
			//go io.Copy(os.Stdout, pr)
			command.Run()
			return false
		})
	})
	port := os.Getenv("PORT")
	router.Run(":" + port)
}
