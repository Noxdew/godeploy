package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Config struct {
	Endpoint        string
	Port            string
	Method          string
	RepoDir         string
	RepoBranch      string
	RepoBranchCheck bool
	RepoRunScript   string
	RepoSecret      string
}

type GitRes struct {
	ref string
}

func main() {

	configFile, err := ioutil.ReadFile("godeploy.conf")
	if err != nil {
		fmt.Println("Failed to find the godeploy.conf: ", err)
		return
	}

	var conf Config

	err = json.Unmarshal(configFile, &conf)
	if err != nil {
		fmt.Println("Failed to parse godeploy.conf: ", err)
		return
	}

	var command = "cd " + conf.RepoDir + " && git checkout " + conf.RepoBranch + " && git pull && " + conf.RepoRunScript

	var cmd = exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}

	http.HandleFunc("/"+conf.Endpoint, func(res http.ResponseWriter, req *http.Request) {
		if req.Method == conf.Method {
			var git GitRes

			if conf.RepoBranchCheck {
				reqBody, err := ioutil.ReadAll(req.Body)
				if err != nil {
					fmt.Println("Failed to read request body: ", err)
					fmt.Fprintf(res, "Failed to verify branch")
					return
				}
				err = json.Unmarshal(reqBody, &git)
				if err != nil {
					fmt.Println("Failed to parse request body: ", err)
					fmt.Fprintf(res, "Failed to verify branch")
					return
				}
			}

			if !conf.RepoBranchCheck || git.ref == "refs/heads/"+conf.RepoBranch {
				for !cmd.ProcessState.Exited() {
					err = cmd.Process.Signal(syscall.SIGINT)
					if err != nil {
						fmt.Println(err)
					}
				}
				cmd = exec.Command("sh", "-c", command)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Start()
				if err != nil {
					fmt.Println(err)
				}
			}
			fmt.Fprintf(res, "Done")
		} else {
			fmt.Println("Received ", req.Method, " request on the deployment endpoint")
		}
	})

	s := &http.Server{
		Addr:           ":" + conf.Port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Println("Listening on port ", conf.Port, " for deployment requests")
	fmt.Println(s.ListenAndServe())
}
