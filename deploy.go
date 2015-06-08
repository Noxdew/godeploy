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
	Endpoint      string
	Port          string
	Method        string
	RepoDir       string
	RepoBranch    string
	RepoRunScript string
	RepoSecret    string
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
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
	}

	http.HandleFunc("/"+conf.Endpoint, func(res http.ResponseWriter, req *http.Request) {
		if req.Method == conf.Method {
			fmt.Println(req.FormValue("ref"))
			if req.FormValue("ref") == "refs/heads/"+conf.RepoBranch {
				for !cmd.ProcessState.Exited() {
					err = cmd.Process.Signal(syscall.SIGINT)
					if err != nil {
						fmt.Println(err)
					}
				}
				cmd = exec.Command("sh", "-c", command)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					fmt.Println(err)
				}
			}
			fmt.Fprintf(res, "Done")
		}
	})

	s := &http.Server{
		Addr:           ":" + conf.Port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Println(s.ListenAndServe())
}
