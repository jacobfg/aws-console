package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

var (
	awsRegion = "ap-southeast-2"
)

type SignInToken struct {
	SigninToken string `json:"SigninToken"`
}

//Console Federated login
type SessionCredentials struct {
	AccessKeyId     string `json:"sessionId"`
	SecretAccessKey string `json:"sessionKey"`
	SessionToken    string `json:"sessionToken"`
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func expandPath(paths ...string) string {
	return os.ExpandEnv(filepath.Join(paths...))
}

func (s *SignInToken) getSigninURL() string {
	req, _ := http.NewRequest("GET", "https://signin.aws.amazon.com/federation", nil)
	// if err != nil {
	// 	log.Debug().Err(err).Msg("Error build request object")
	// 	return "", errors.New("Error build request object")
	// }
	// if you appending to existing query this works fine
	q := req.URL.Query()
	// or you can create new url.Values struct and encode that like so
	q.Add("Action", "login")
	q.Add("Destination", fmt.Sprintf("https://us-east-2.console.aws.amazon.com/console/home?region=%s", awsRegion))
	q.Add("SigninToken", s.SigninToken)
	req.URL.RawQuery = q.Encode()
	return req.URL.String()
}

func getConsoleSigninUrl(duration int) (string, error) {
	// log := log.With().Str("profile", c.Name).Logger()
	client := http.Client{}

	fmt.Println(expandPath("$HOME", ".aws", "credentials"))
	creds := credentials.NewSharedCredentials(expandPath("$HOME", ".aws", "credentials"), getEnv("AWS_PROFILE", "default"))
	// creds := credentials.NewEnvCredentials()

	// Retrieve the credentials value
	credValue, err := creds.Get()
	if err != nil {
		// handle error
		return "", errors.New("Failed to get AWS credentials from environment")
	}

	sessionCreds := &SessionCredentials{
		AccessKeyId:     credValue.AccessKeyID,
		SecretAccessKey: credValue.SecretAccessKey,
		SessionToken:    credValue.SessionToken,
	}

	jdata, err := json.Marshal(sessionCreds)
	if err != nil {
		// log.Debug().Err(err).Msg("Error marshalling credentials into JSON object")
		return "", errors.New("error marshalling credentials into JSON object")
	}
	fmt.Print(string(jdata))

	req, err := http.NewRequest("GET", "https://signin.aws.amazon.com/federation", nil)
	if err != nil {
		// log.Debug().Msg("Error building session request")
		return "", errors.New("error building session request")
	}

	// build query for getting a session url
	q := req.URL.Query()
	q.Add("Action", "getSigninToken")
	q.Add("Session", string(jdata))
	if duration > 0 {
		q.Add("SessionDuration", strconv.FormatInt(int64(duration), 10))
	}
	req.URL.RawQuery = q.Encode()

	// log.Debug().Msg("Sending AWS SignIn Token request")
	resp, _ := client.Do(req)

	// check response code as 200
	if resp.StatusCode != 200 {
		// log.Debug().Msg("Failed to get AWS Signin Token")
		return "", errors.New("failed to get AWS Signin Token")
	}

	// defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var signinToken SignInToken
	err = json.Unmarshal(body, &signinToken)
	if err != nil {
		// log.Debug().Err(err).Msg("Error unmarshalling signin token")
		return "", errors.New("error unmarshalling signin token")
		// } else {
		// 	log.Debug().Str("token", signinToken.SigninToken).Msg("Signin Token")
	}

	// ------ //
	return signinToken.getSigninURL(), nil
}

func main() {
	url, err := getConsoleSigninUrl(43200) // 12 hours
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(url)
}
