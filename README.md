[![Circle CI](https://circleci.com/gh/morwoen/godeploy.svg?style=svg)](https://circleci.com/gh/morwoen/godeploy)
# GoDeploy
A simple to use tool for automatic deployment of projects.

The configuration is as follows:
```json
{
	"ServerEndpoint": "deploy",
	"ServerPort": "1313",
	"ServerMethod": "POST",
	"RepoBranch": "master",
	"RepoBranchCheck": true,
	"RepoBuildScript": "",
	"RepoRunScript": "make",
	"RepoSecret": "",
	"RepoSecretAlg": "sha1",
	"RepoSecretHeader": "X_Hub_Signature",
	"RepoSecretPrefix": true,
	"ScriptDir": "",
	"ScriptAlwaysWait": false,
	"ScriptRunAtStart": true
}
```

`ServerEndpoint`: The path of the endpoint it should listen to for requests. Do **NOT** include a `/` at the beginning. Empty string will listen to `http://localhost`

`ServerPort`: The port it should listen to. Remember that it should be free!

`ServerMethod`: The method it should listen for. The default for GitHub is POST.

`RepoBranch`: The name of the git repository's branch it should deploy

`RepoBranchCheck`: Whether it should check that the request body has `"ref": "refs/heads/{branchname}"`

`RepoBuildScript`: The sceript it should use when it is inside the repository folder.

`RepoRunScript`: The script to run the project, again from the repository folder.

`RepoSecret`: The secret key to check the origin of the request.

`RepoSecretAlg`: The type of algorithm to use for the request hash. Currently supported `sha1`, `sha256` and `sha512`.

`RepoSecretHeader`: The name of the header containing the hash of the body.

`RepoSecretPrefix`: Whether there is a prefix in front of the hash. In GitHub there is a prefix (`sha1=`)

`ScriptDir`: The directory for the run script. They are joined at run time. Example: `/home/project/bin` with run script `../run.sh` will execute `/home/project/run.sh`

`ScriptAlwaysWait`: Whether to wait for the script to finish before starting a new deployment.

`ScriptRunAtStart`: Whether to start the project when you start GoDeploy.


NOTE: The config file must be in either the folder with the executable or the current folder you are running the application with

# TODO
1. Make the update script configurable so that it can be used with non-git repos
2. Add option for https
3. Remember to restart the script when `"ScriptAlwaysWait": true`
4. Testing
