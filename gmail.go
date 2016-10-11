package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// ListAccounts - list of accounts from config file
var ListAccounts []Account

// Account type - description of account
type Account struct {
	Account  string `json:"account"`
	Short    string `json:"short_conky"`
	Email    string `json:"email"`
	Password string `json:"password"`
	URL      string
}

// Error function
func check(e error) {
	if e != nil {
		log.Fatal(e.Error())
	}
}

func init() {
	filename := fmt.Sprintf("%s/.gmail.json", os.Getenv("HOME"))
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		// if file with configuration does`nt exists this part will create it
		f, err := os.Create(filename)
		check(err)
		defer f.Close()
		exampleAccount := Account{
			Account:  "ACCOUNT",
			Short:    "SHORT",
			Email:    "EMAIL@gmail.com",
			Password: "PASSWORD",
		}
		listAccounts := []Account{exampleAccount}
		exampleJSON, err := json.Marshal(listAccounts)
		check(err)
		f.WriteString(string(exampleJSON))
		ListAccounts = listAccounts
	} else {
		listAccounts := &ListAccounts
		err := json.Unmarshal(content, listAccounts)
		check(err)
	}
}

func grep(str string) string {
	r, err := regexp.Compile(`<fullcount>(.*?)</fullcount>`)
	check(err)
	substring := r.FindString(str)
	re, err := regexp.Compile(`[\d+]`)
	check(err)
	return re.FindString(substring)
}

// getMailCount - new goroutine for checking emails
func getMailCount(channel chan<- string, acc Account) {
	resp, err := http.Get(acc.URL)
	check(err)
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	count := grep(string(body))
	channel <- fmt.Sprintf("%[1]v:%[2]v ", acc.Short, count)
}

func main() {
	baseURL := "https://{{.Email}}:{{.Password}}@mail.google.com/mail/feed/atom"
	// Check if domain online
	resp, err := http.Get("https://mail.google.com")
	if err == nil || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMovedPermanently {
		channel := make(chan string)
		defer close(channel)
		for _, acc := range ListAccounts {
			t := template.New(acc.Account)
			t, err := t.Parse(baseURL)
			check(err)
			buf := new(bytes.Buffer)
			err = t.Execute(buf, acc)
			check(err)
			acc.URL = buf.String()
			// separate all network requests to goroutines
			go getMailCount(channel, acc)
		}
		accLen := len(ListAccounts)
		counts := make([]string, accLen)
		for i := 0; i < accLen; i++ {
			counts[i] = <-channel
		}
		fmt.Println(strings.Join(counts, ""))
	}
}
