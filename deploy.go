package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"hash"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	ServerEndpoint   string
	ServerPort       string
	ServerMethod     string
	RepoBranch       string
	RepoBuildScript  string
	RepoRunScript    string
	RepoSecret       string
	RepoSecretAlg    string
	RepoSecretHeader string
	RepoSecretPrefix bool
	RepoBranchCheck  bool
	ScriptDir        string
	ScriptAlwaysWait bool
	ScriptRunAtStart bool
}

type GitRes struct {
	Ref string
}

func CheckMAC(message, messageMAC, key []byte, hashFunc func() hash.Hash) bool {
	mac := hmac.New(hashFunc, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

func main() {

	execFolder, err := os.Getwd()
	if err != nil {
		fmt.Println("GoDeploy: Failed to get the executable path")
	}
	configFile, err := ioutil.ReadFile(filepath.Clean(execFolder + "/" + "godeploy.conf"))
	if err != nil {
		fmt.Println("GoDeploy: Failed to find the godeploy.conf (trying again): ", err)
		configFile, err = ioutil.ReadFile("godeploy.conf")
		if err != nil {
			fmt.Println("GoDeploy: Failed to find the godeploy.conf: ", err)
			return
		}
	}

	var conf Config

	err = json.Unmarshal(configFile, &conf)
	if err != nil {
		fmt.Println("GoDeploy: Failed to parse godeploy.conf: ", err)
		return
	}

	if conf.ScriptDir == "" {
		conf.ScriptDir = execFolder
	}

	var hashFunc func() hash.Hash

	if strings.EqualFold(conf.RepoSecretAlg, "sha512") {
		hashFunc = sha512.New
	} else if strings.EqualFold(conf.RepoSecretAlg, "sha256") {
		hashFunc = sha256.New
	} else {
		hashFunc = sha1.New
	}

	if conf.RepoSecret == "" {
		conf.RepoSecret = os.Getenv("REPO_SECRET")
	}

	var updateCommand = "cd " + execFolder + " && git checkout " + conf.RepoBranch + " && git pull"
	var buildCommand = strings.Fields(filepath.Clean(conf.ScriptDir + "/" + conf.RepoBuildScript))
	var command = strings.Fields(filepath.Clean(conf.ScriptDir + "/" + conf.RepoRunScript))
	var errChan = make(chan error)

	var tempCommand []string
	if conf.ScriptRunAtStart {
		tempCommand = command
	} else {
		tempCommand = []string{""}
	}

	var cmd = exec.Command("sh", "-c", updateCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("GoDeploy: Failed to update the repository", err)
	}

	cmd = exec.Command(buildCommand[0], buildCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("GoDeploy: Failed to build the project", err)
	}

	cmd = exec.Command(tempCommand[0], tempCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		errChan <- cmd.Run()
	}()

	http.HandleFunc("/"+conf.ServerEndpoint, func(res http.ResponseWriter, req *http.Request) {
		if strings.EqualFold(req.Method, conf.ServerMethod) {
			// var git GitRes
			var correctBranch = false
			if conf.RepoBranchCheck || conf.RepoSecret != "" {
				reqBody, err := ioutil.ReadAll(req.Body)
				if err != nil {
					fmt.Println("GoDeploy: Failed to read request body: ", err)
					fmt.Fprintf(res, "GoDeploy: Failed to verify branch")
					return
				}

				if conf.RepoSecret != "" {
					fmt.Println("Sha header: ", req.Header)
					hashed := req.Header.Get(conf.RepoSecretHeader)
					if conf.RepoSecretPrefix {
						spliced := strings.Split(hashed, "=")
						hashed = spliced[len(spliced)-1]
					}
					if !CheckMAC(reqBody, []byte(hashed), []byte(conf.RepoSecret), hashFunc) {
						fmt.Println("GoDeploy: Failed to verify request origin")
						fmt.Fprintf(res, "GoDeploy: Failed to verify origin")
						return
					}
				}

				// if conf.RepoBranchCheck {
				// 	err = json.Unmarshal(reqBody, &git)
				// 	if err != nil {
				// 		fmt.Println("GoDeploy: Failed to parse request body: ", err)
				// 		fmt.Fprintf(res, "GoDeploy: Failed to verify branch")
				// 		return
				// 	}
				// }
				correctBranch = strings.Contains(string(reqBody), "\"" + conf.RepoBranch + "\"")
			}

			if !conf.RepoBranchCheck || correctBranch {

				var temoCmd = exec.Command("sh", "-c", updateCommand)
				temoCmd.Stdout = os.Stdout
				temoCmd.Stderr = os.Stderr
				err = temoCmd.Run()
				if err != nil {
					fmt.Println("GoDeploy: Failed to update the repository", err)
				}

				temoCmd = exec.Command(buildCommand[0], buildCommand[1:]...)
				temoCmd.Stdout = os.Stdout
				temoCmd.Stderr = os.Stderr
				err = temoCmd.Run()
				if err != nil {
					fmt.Println("GoDeploy: Failed to build the project", err)
				}

				select {
				default:
					fmt.Println("GoDeploy: Process still running")
					if !conf.ScriptAlwaysWait {
						err = cmd.Process.Signal(os.Interrupt)
						if err != nil {
							fmt.Println(err)
						}
					}
					ps, err := cmd.Process.Wait()
					if err != nil {
						fmt.Println(err)
					}
					if ps != nil && ps.Exited() {
						fmt.Println("GoDeploy: Process exited")
					} else {
						fmt.Println("GoDeploy: Process almost finished or no process state")
					}
					if err = <-errChan; err != nil {
						fmt.Printf("GoDeploy: Process exited: %s\n", err)
					} else {
						fmt.Println("GoDeploy: Process exited without error")
					}
				case e := <-errChan:
					if e != nil {
						fmt.Printf("GoDeploy: Process exited: %s\n", e)
					} else {
						fmt.Println("GoDeploy: Process exited without error")
					}
				}

				cmd = exec.Command(command[0], command[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				go func() {
					errChan <- cmd.Run()
				}()
			} else {
				fmt.Println("GoDeploy: Push to different branch")
			}
			fmt.Fprintf(res, "GoDeploy: Done")
		} else {
			fmt.Println("GoDeploy: Received ", req.Method, " request on the deployment endpoint")
		}
	})

	s := &http.Server{
		Addr:           ":" + conf.ServerPort,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Println("GoDeploy: Listening on port ", conf.ServerPort, " for deployment requests")
	fmt.Println("GoDeploy: ", s.ListenAndServe())
}
