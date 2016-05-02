package cronbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/", doHello)
	http.HandleFunc("/minutes/remind", doMinutesRemind)
}

func httpError(ctx context.Context, w http.ResponseWriter, r *http.Request, s string, args ...interface{}) {
	msg := fmt.Sprintf(s, args...)
	log.Errorf(ctx, msg)
	http.Error(w, msg, http.StatusInternalServerError)
}

func doHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World!")
}

func doMinutesRemind(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	endpoint := os.Getenv("SLACKGW_ENDPOINT")
	if endpoint == "" {
		httpError(ctx, w, r, "missing SLACKGW_ENDPOINT")
		return
	}

	token := os.Getenv("SLACKGW_TOKEN")
	if token == "" {
		httpError(ctx, w, r, "missing SLACKGW_TOKEN")
		return
	}

	githubtoken := os.Getenv("GITHUB_TOKEN")
	if githubtoken == "" {
		httpError(ctx, w, r, "missing GITHUB_TOKEN")
		return
	}

	client := urlfetch.Client(ctx)

	now := time.Now()
	localpath := now.Format("minutes/2006/0102.md")
	githuburl := "https://api.github.com/repos/builderscon/planning/contents/" + localpath
	buf := bytes.Buffer{}
	json.NewEncoder(&buf).Encode(map[string]interface{}{
		"message": "create a new minutes",
		"committer": map[string]interface{}{
			"name":  "builderscon cronbot",
			"email": "builderscon.io@gmail.com",
		},
		"content": "# Minutes (" + now.Format("2006/01/02") + ")\n" +
			"\n" +
			"Please read the README for the format of these minutes\n\n" +
			"## lestrrat\n",
	})
	req, err := http.NewRequest("PUT", githuburl, &buf)
	if err != nil {
		httpError(ctx, w, r, "failed to create request to create new minutes: %s", err)
		return
	}
	req.Header.Set("Authorization", "Basic "+githubtoken)
	req.Header.Set("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		httpError(ctx, w, r, "failed to make HTTP request: %s", err)
		return
	}

	buf.Reset()
	buf.WriteString(
		(url.Values{
			"message": []string{
				"月曜日です！祝日じゃなければ週報を書いてくださいね！\n" +
					"https://github.com/builderscon/planning/tree/master/" + localpath,
			},
			"channel": []string{"#random"},
		}).Encode(),
	)

	req, err = http.NewRequest("POST", endpoint+"/post", &buf)
	if err != nil {
		httpError(ctx, w, r, "failed to create HTTP request: %s", err)
		return
	}
	req.Header.Set("X-Slackgw-Auth", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		httpError(ctx, w, r, "failed to make HTTP request: %s", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		httpError(ctx, w, r, "received invalid response: %s", res.Status)
		return
	}

	w.WriteHeader(http.StatusOK)
}