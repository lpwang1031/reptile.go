package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	imagetItemExp = regexp.MustCompile(`src="//i\.4cdn\.org/s/[0123456789]+s\.(jpg|jpeg|png|gif)"`)
	threadItemExp = regexp.MustCompile(`"thread/[0123456789]+"`)
	ch            = make(chan int)
)

type ThreadItem struct {
	url     string
	content string
	imgs    []string
}

func (t *ThreadItem) getContent() *ThreadItem {
	content, err := httpGet(t.url)
	if err != 200 {
		t.content = ""
		return t
	}

	t.content = string(content)
	return t
}

func (t *ThreadItem) getImage() *ThreadItem {
	imgs := imagetItemExp.FindAllStringSubmatch(t.content, 10*1000)
	l := make([]string, 0)
	for _, v := range imgs {
		l = append(l, v[0])
	}
	t.imgs = l
	return t
}

func (t *ThreadItem) download() {
	last := strings.LastIndex(t.url, "/") + 1
	pwd, _ := os.Getwd()
	dir := pwd + "\\download\\" + string(t.url[last:len(t.url)])
	fmt.Println("create dir:", dir)

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, img := range t.imgs {
		pos := strings.LastIndex(img, "/")
		filename := string(img[pos : len(img)-1])
		file, err := os.Create(dir + "/" + filename)
		if err != nil {
			fmt.Println("创建文件失败:", err)
			continue
		}
		defer file.Close()
		data, error := downloadImg("http:" + string(img[5:len(img)-1]))
		if error != 200 {
			fmt.Println("下载图片失败", error)
		}
		file.Write(data)
	}
}

func findThreads(url string) []ThreadItem {
	var threads = make([]ThreadItem, 0)
	content, err := httpGet(url)
	if err != 200 {
		return threads
	}
	tds := threadItemExp.FindAllStringSubmatch(content, 10*1000)
	var tdsr = make([]string, 0)

	for _, t := range tds {
		var n = strings.Replace(t[0], "\"", "", -1)
		tdsr = append(tdsr, n)
	}
	sort.Strings(tdsr)
	tdsr = unequal(tdsr)

	for _, t := range tdsr {
		threads = append(threads, ThreadItem{url: "http://boards.4chan.org/s/" + t})
	}
	return threads
}

func httpGet(url string) (contene string, statusCode int) {
	resp, err := http.Get(url)
	if err != nil {
		statusCode = -100
		return
	}
	defer resp.Body.Close()
	data, err2 := ioutil.ReadAll(resp.Body)

	if err2 != nil {
		statusCode = -200
		return
	}

	statusCode = resp.StatusCode
	contene = string(data)
	return
}

func downloadImg(url string) (content []byte, statusCode int) {
	// url = strings.Replace(url, "s.", ".", -1)
	fmt.Println("download img from url:", url)
	resp, err1 := http.Get(url)
	if err1 != nil {
		statusCode = -100
		return
	}
	if resp.StatusCode == 404 {
		url = strings.Replace(url, ".jpg", ".png", -1)
		resp, err1 = http.Get(url)
		if err1 != nil {
			statusCode = -100
			return
		}
	}
	defer resp.Body.Close()
	content, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		statusCode = -200
		return
	}
	statusCode = resp.StatusCode
	return
}

func unequal(a []string) (ret []string) {
	alen := len(a)
	for i := 0; i < alen; i++ {
		if i > 0 && a[i-1] == a[i] {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

func work(url string, ch chan int) {
	fmt.Println("get list with url", url)
	var threads = findThreads(url)
	for _, v := range threads {
		(&v).getContent().getImage().download()
	}
	ch <- 1
}

func main() {
	pages := []string{"4", "5"}
	for _, index := range pages {
		go work("http://boards.4chan.org/s/"+index+"/", ch)
	}
	<-ch
	<-ch
}
