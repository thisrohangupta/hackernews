package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Story struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	By    string `json:"by"`
	Score int    `json:"score"`
	URL   string `json:"url"`
	Time  int64  `json:"time"`
}

type PageData struct {
	Stories template.HTML
}

func fetchTopStories(limit int) ([]Story, error) {
	resp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}
	stories := make([]Story, 0, limit)
	for i, id := range ids {
		if i >= limit {
			break
		}
		story, err := fetchStory(id)
		if err == nil && story.Title != "" {
			stories = append(stories, story)
		}
	}
	return stories, nil
}

func fetchStory(id int) (Story, error) {
	resp, err := http.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
	if err != nil {
		return Story{}, err
	}
	defer resp.Body.Close()
	var story Story
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return Story{}, err
	}
	return story, nil
}

func renderStoriesHTML(stories []Story) template.HTML {
	html := ""
	for _, s := range stories {
		storyURL := s.URL
		if storyURL == "" {
			storyURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", s.ID)
		}
		timeStr := time.Unix(s.Time, 0).Format("Jan 2, 2006 15:04")
		html += fmt.Sprintf(
			`<div class="story">
				<div class="story-title"><a href="%s" target="_blank">%s</a></div>
				<div class="story-meta">by %s | %d points | %s</div>
			</div>`,
			storyURL, template.HTMLEscapeString(s.Title), template.HTMLEscapeString(s.By), s.Score, timeStr,
		)
	}
	return template.HTML(html)
}

func main() {
	stories, err := fetchTopStories(20)
	if err != nil {
		fmt.Println("Error fetching stories:", err)
		return
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Println("Error loading template:", err)
		return
	}

	os.MkdirAll("public", 0755)
	f, err := os.Create(filepath.Join("public", "index.html"))
	if err != nil {
		fmt.Println("Error creating index.html:", err)
		return
	}
	defer f.Close()

	data := PageData{
		Stories: renderStoriesHTML(stories),
	}
	err = tmpl.Execute(f, data)
	if err != nil {
		fmt.Println("Error rendering template:", err)
		return
	}

	// Copy static assets
	copyStatic("static", "public/static")

	fmt.Println("Site generated! Open public/index.html in your browser.")
}

func copyStatic(src, dst string) {
	os.MkdirAll(dst, 0755)
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		in, err := os.Open(filepath.Join(src, file.Name()))
		if err != nil {
			continue
		}
		defer in.Close()
		out, err := os.Create(filepath.Join(dst, file.Name()))
		if err != nil {
			continue
		}
		defer out.Close()
		io.Copy(out, in)
	}
}
