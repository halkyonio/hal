package io

import (
	"archive/zip"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"halkyon.io/hal/pkg/log"
	"io"
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func Generate(url, name string) error {
	body := get(url)

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	zipFile := filepath.Join(currentDir, name+".zip")
	err = ioutil.WriteFile(zipFile, body, 0644)
	if err != nil {
		return fmt.Errorf("failed to download file %s due to %s", zipFile, err)
	}
	// output zipped file into proper child directory
	dir := filepath.Join(filepath.Dir(zipFile), name)
	err = Unzip(zipFile, dir)
	if err != nil {
		return fmt.Errorf("failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		return err
	}
	return nil
}

func get(url string) []byte {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	LogErrorAndExit(err, "error creating request for "+url)
	addClientHeader(req)

	res, err := client.Do(req)
	if err == nil && res.StatusCode >= 400 {
		msg := fmt.Sprintf("server returned a '%s' error", res.Status)
		if res.Body != nil {
			if bytes, err := ioutil.ReadAll(res.Body); err == nil {
				msg = msg + ": " + string(bytes)
			}
		}

		err = fmt.Errorf(msg)
	}
	LogErrorAndExit(err, fmt.Sprintf("error performing request to %v", req.URL))

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	LogErrorAndExit(err, fmt.Sprintf("error reading response body %v", res))

	if strings.Contains(string(body), "Application is not available") {
		logrus.Fatal("Generator service is not available")
	}

	return body

}

func HttpGet(url, endpoint string, values *url.Values) []byte {
	u := strings.Join([]string{url, endpoint}, "/")
	if values != nil {
		parameters := values.Encode()
		if len(parameters) > 0 {
			u = u + "?" + parameters
		}
	}

	return get(u)
}

func UnmarshallYamlFromHttp(url, endpoint string, result interface{}) {
	body := HttpGet(url, endpoint, nil)
	unmarshall(body, result)
}

func unmarshall(body []byte, result interface{}) {
	err := yaml.Unmarshal(body, &result)
	if err != nil {
		LogErrorAndExit(err, "error unmarshalling")
	}
}

func addClientHeader(req *http.Request) {
	userAgent := "halkyon-hal/1.0"
	req.Header.Set("User-Agent", userAgent)
}

// LogErrorAndExit prints the cause of the given error and exits the code with an
// exit code of 1.
// If the context is provided, then that is printed, if not, then the cause is
// detected using errors.Cause(err)
func LogErrorAndExit(err error, context string, a ...interface{}) {
	if err != nil {
		msg := errors.Cause(err).Error()
		switch t := err.(type) {
		case k8serrors.APIStatus:
			reason := k8serrors.ReasonForError(err)
			msg = fmt.Sprintf("error communicating with cluster: %s", reason)
		default:
			errName := reflect.TypeOf(t).Name()
			if len(errName) > 0 {
				msg = fmt.Sprintf("%s: %s", errName, msg)
			} else {
				msg = fmt.Sprintf("%s", msg)
			}
		}

		if len(context) == 0 {
			log.Error(msg)
		} else {
			log.Errorf(fmt.Sprintf("%s: %s", context, msg), a...)
		}
		os.Exit(1)
	}
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		name := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(name, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(name, string(os.PathSeparator)); lastIndex > -1 {
				fdir = name[:lastIndex]
			}

			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(
				name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
