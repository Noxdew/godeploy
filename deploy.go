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
	ServerEndpoint   string
	ServerPort       string
	ServerMethod     string
	RepoDir          string
	RepoBranch       string
	RepoRunScript    string
	RepoSecret       string
	RepoBranchCheck  bool
	ScriptAlwaysWait bool
	ScriptRunAtStart bool
}

type GitRes struct {
	Ref string
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
	var errChan = make(chan error)

	var tempCommand string
	if conf.ScriptRunAtStart {
		tempCommand = command
	} else {
		tempCommand = ""
	}

	var cmd = exec.Command("sh", "-c", tempCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		errChan <- cmd.Run()
	}()

	http.HandleFunc("/"+conf.ServerEndpoint, func(res http.ResponseWriter, req *http.Request) {
		if req.Method == conf.ServerMethod {
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

			if !conf.RepoBranchCheck || git.Ref == "refs/heads/"+conf.RepoBranch {
				for !cmd.ProcessState.Exited() {
					err = cmd.Process.Signal(syscall.SIGINT)
					if err != nil {
						fmt.Println(err)
					}
					select {
					default:
						fmt.Println("process still running")
					case e := <-errChan:
						if e != nil {
							fmt.Printf("process exited: %s\n", e)
						} else {
							fmt.Println("process exited without error")
						}
					}
				}
				cmd = exec.Command("sh", "-c", command)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				go func() {
					errChan <- cmd.Run()
				}()
			}
			fmt.Fprintf(res, "Done")
		} else {
			fmt.Println("Received ", req.Method, " request on the deployment endpoint")
		}
	})

	s := &http.Server{
		Addr:           ":" + conf.ServerPort,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Println("Listening on port ", conf.ServerPort, " for deployment requests")
	fmt.Println(s.ListenAndServe())
}
