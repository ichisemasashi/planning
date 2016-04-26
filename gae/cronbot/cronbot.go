package cronbot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/minutes/remind", doMinutesRemind)
}

func doMinutesRemind(w http.ResponseWriter, r *http.Request) {
	endpoint := os.Getenv("SLACKGW_ENDPOINT")
	if endpoint == "" {
		http.Error(w, "missing SLACKGW_ENDPOINT", http.StatusInternalServerError)
		return
	}

	token := os.Getenv("SLACKGW_TOKEN")
	if token == "" {
		http.Error(w, "missing SLACKGW_TOKEN", http.StatusInternalServerError)
		return
	}

	githubtoken := os.Getenv("GITHUB_TOKEN")
	if githubtoken == "" {
		http.Error(w, "missing GITHUB_TOKEN", http.StatusInternalServerError)
		return
	}

	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	now := time.Now()
	localpath := now.Format("minutes/2006/0102.md")
	path := "/repos/builderscon/planning/contents/" + localpath
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
	req, err := http.NewRequest("PUT", path, &buf)
	if err != nil {
		http.Error(w, "failed to create request to create new minutes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Basic "+githubtoken)
	req.Header.Set("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		http.Error(w, "failed to make HTTP request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	buf.Reset()
	buf.WriteString(
		(url.Values{
			"message": []string{
				"月曜日です！祝日じゃなければ週報を書いてくださいね！\n"+
				"https://github.com/builderscon/planning/tree/master/" + localpath,
			},
			"channel": []string{"#random"},
		}).Encode(),
	)

	req, err = http.NewRequest("POST", endpoint+"/post", &buf)
	if err != nil {
		http.Error(w, "failed to create HTTP request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("X-Slackgw-Auth", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to make HTTP request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if res.StatusCode != http.StatusOK {
		http.Error(w, "received invalid response: "+res.Status, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}