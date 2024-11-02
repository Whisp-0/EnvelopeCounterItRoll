package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

const (
	apiVersion = "5.199"
	itRollID   = "-218375169"
)

var accessToken string
var totalEnvelopes int

type VKResponse struct {
	Response struct {
		Count int `json:"count"`
		Items []struct {
			ID       int `json:"id"`
			Comments struct {
				Count int `json:"count"`
			} `json:"comments"`
		} `json:"items"`
	} `json:"response"`
}

type VKCommentsResponse struct {
	Response struct {
		Items []struct {
			Text string `json:"text"`
		} `json:"items"`
	} `json:"response"`
}

func getPostsCount() (int, error) {
	url := fmt.Sprintf("https://api.vk.com/method/wall.get?v=%s&access_token=%s&owner_id=%s&count=1", apiVersion, accessToken, itRollID)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var vkResp VKResponse
	if err := json.NewDecoder(resp.Body).Decode(&vkResp); err != nil {
		return 0, err
	}

	return vkResp.Response.Count, nil
}

func getPosts(offset, count int) []int {
	url := fmt.Sprintf("https://api.vk.com/method/wall.get?owner_id=%s&v=%s&access_token=%s&count=%d&offset=%d", itRollID, apiVersion, accessToken, count, offset)
	resp, _ := http.Get(url)
	defer resp.Body.Close()

	var vkResp VKResponse
	if err := json.NewDecoder(resp.Body).Decode(&vkResp); err != nil {
		return nil
	}

	var postIDs []int
	for _, item := range vkResp.Response.Items {
		if item.Comments.Count > 0 {
			postIDs = append(postIDs, item.ID)
		}
	}
	return postIDs
}

func processPost(postID int, wg *sync.WaitGroup, m *sync.Mutex) {
	defer wg.Done()

	url := fmt.Sprintf("https://api.vk.com/method/wall.getComments?v=%s&access_token=%s&owner_id=%s&post_id=%d", apiVersion, accessToken, itRollID, postID)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var vkResp VKCommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&vkResp); err != nil {
		return
	}

	m.Lock()
	for _, comment := range vkResp.Response.Items {
		totalEnvelopes += strings.Count(strings.ToLower(comment.Text), "энвилоуп")
	}
	m.Unlock()
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	accessToken = os.Getenv("ACCESS_TOKEN")
	start := time.Now()

	postsCount, _ := getPostsCount()

	step := 20
	var wg sync.WaitGroup
	postIDs := make(chan int, postsCount)
	for i := 0; i < postsCount; i += step {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			ids := getPosts(offset, step)
			for _, id := range ids {
				postIDs <- id
			}
		}(i)

	}
	go func() {
		wg.Wait()
		close(postIDs)
	}()

	var processWg sync.WaitGroup
	mu := sync.Mutex{}
	for postID := range postIDs {
		processWg.Add(1)
		go processPost(postID, &processWg, &mu)

	}

	processWg.Wait()
	fmt.Println("Всего эвилоупов:", totalEnvelopes)

	elapsed := time.Since(start)
	fmt.Println(elapsed)
}
