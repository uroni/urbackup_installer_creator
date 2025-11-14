// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ncruces/zenity"
	"golang.org/x/crypto/pbkdf2"
)

type ServerSettings struct {
	Url string
}

func get_response(server_settings ServerSettings, action string, params url.Values, method string) (resp *http.Response, err error) {

	var get_params string
	var request_body *strings.Reader
	if method == "GET" {
		request_body = strings.NewReader("")
		get_params = "&" + params.Encode()
	} else {
		request_body = strings.NewReader(params.Encode())
		get_params = ""
	}

	req, err := http.NewRequest(method, server_settings.Url+"?a="+action+get_params,
		request_body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func get_json(server_settings ServerSettings, action string, params url.Values) (json string, err error) {

	resp, err := get_response(server_settings, action, params, "POST")

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

type SaltResp struct {
	Ses           string
	Salt          string
	Pbkdf2_rounds int
	Rnd           string
	Error         int
}

func get_salt(server_settings ServerSettings, username string) (sr *SaltResp, err error) {
	setStatus("Getting login information from server...")

	json_str, err := get_json(server_settings, "salt", url.Values{"username": {username}})

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(json_str), &sr)
	if err != nil {
		return nil, err
	}

	if sr.Error != 0 || len(sr.Salt) == 0 {
		if sr.Error == 0 {
			return nil, errors.New("User not found on server")
		}
		return nil, errors.New("Error getting salt")
	}

	return sr, nil
}

type LoginResp struct {
	Success bool
	Error   int
}

func login(server_settings ServerSettings, username string, sr *SaltResp, password string) error {

	setStatus("Logging into server...")

	password_md5_bin := md5.Sum([]byte(sr.Salt + password))
	password_md5 := hex.EncodeToString(password_md5_bin[:])

	if sr.Pbkdf2_rounds > 0 {
		password_md5 = hex.EncodeToString(pbkdf2.Key(password_md5_bin[:],
			[]byte(sr.Salt), sr.Pbkdf2_rounds, 32, sha256.New))
	}

	password_md5_bin = md5.Sum([]byte(sr.Rnd + password_md5))
	password_md5 = hex.EncodeToString(password_md5_bin[:])

	json_str, err := get_json(server_settings, "login", url.Values{"username": {username},
		"password": {password_md5},
		"ses":      {sr.Ses}})

	if err != nil {
		return err
	}

	var lr LoginResp
	err = json.Unmarshal([]byte(json_str), &lr)
	if err != nil {
		return err
	}

	if lr.Error != 0 || !lr.Success {
		return errors.New("Error logging in")
	}

	return nil
}

type StatusClientDownload struct {
	Name string
	Id   int
}

type StatusResp struct {
	Client_downloads []StatusClientDownload
	Error            int
}

func get_status(server_settings ServerSettings, sr *SaltResp) (status *StatusResp, err error) {
	json_str, err := get_json(server_settings, "status", url.Values{"ses": {sr.Ses}})

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(json_str), &status)
	if err != nil {
		return nil, err
	}

	if status.Error != 0 {
		return nil, errors.New("Session timeout")
	}

	return status, nil
}

type AddClientResp struct {
	Already_exists bool
	New_authkey    string
	New_clientid   int
	Error          int
}

func add_client(server_settings ServerSettings, sr *SaltResp, clientname string,
	group_name string) (resp *AddClientResp, err error) {

	params := url.Values{"ses": {sr.Ses}, "clientname": {clientname}}

	if len(group_name) > 0 {
		params.Add("group_name", group_name)
	}

	json_str, err := get_json(server_settings, "add_client", params)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(json_str), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != 0 {
		return nil, errors.New("Session timeout")
	}

	return resp, nil
}

func download_client(server_settings ServerSettings, sr *SaltResp, clientid int, authkey string, tmpdir string, installer_name string, os_linux bool) (file *os.File, err error) {

	setStatus(fmt.Sprintf("Starting download of client id %d", clientid))

	file, err = os.Create(path.Join(tmpdir, installer_name))

	if err != nil {
		return nil, err
	}

	file_fn := file.Name()

	params := url.Values{"ses": {sr.Ses},
		"clientid": {strconv.Itoa(clientid)}}

	if len(authkey) > 0 {
		params.Add("authkey", authkey)
	}
	if os_linux {
		params.Add("os", "linux")
	}

	resp, err := get_response(server_settings, "download_client", params, "GET")

	if err != nil {
		file.Close()
		os.Remove(file_fn)
		return nil, err
	}

	defer resp.Body.Close()

	var limit int64
	limit = 60 * 1024 * 1024
	if os_linux {
		limit = 25 * 1024 * 1024
	}

	setStatus("Downloading installer...")

	// Track progress
	buf := make([]byte, 32*1024)
	var downloaded int64 = 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				file.Close()
				os.Remove(file_fn)
				return nil, writeErr
			}
			downloaded += int64(n)

			// Update progress bar
			percent := int(float64(downloaded) / float64(limit) * 100)
			if percent > 100 {
				percent = 100
			}
			setProgress(percent)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			file.Close()
			os.Remove(file_fn)
			return nil, err
		}
	}

	setStatus("Download complete!")
	return file, nil
}

func mod_notray_write(program_files string) error {
	if _, err := os.Stat(path.Join(program_files, "UrBackup", "UrBackupClientBackend.exe")); os.IsNotExist(err) {
		os.MkdirAll(path.Join(program_files, "UrBackup"), 0744)
		err = ioutil.WriteFile(path.Join(program_files, "UrBackup", "UrBackupClientBackend.exe"), []byte("foo"), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func mod_notray() error {
	program_files := os.Getenv("ProgramW6432")
	if len(program_files) > 0 {
		return mod_notray_write(program_files)
	} else if program_files = os.Getenv("ProgramFiles(x86)"); len(program_files) > 0 {
		return mod_notray_write(program_files)
	}
	return nil
}

func unhex(hexstr string) string {
	ret, _ := hex.DecodeString(hexstr)
	return string(ret)
}

// GUI manager for displaying status in a persistent window
type GUIManager struct {
	progressDlg zenity.ProgressDialog
}

var guiManager *GUIManager

func initGUI() error {
	dlg, err := zenity.Progress(
		zenity.Title("UrBackup Installer"),
		zenity.Width(500),
		zenity.NoCancel(),
	)
	if err != nil {
		return err
	}

	guiManager = &GUIManager{
		progressDlg: dlg,
	}

	return nil
}

func closeGUI() {
	if guiManager != nil && guiManager.progressDlg != nil {
		guiManager.progressDlg.Close()
	}
}

func setStatus(message string) {
	if guiManager == nil {
		return
	}
	guiManager.progressDlg.Text(message)
}

func setProgress(percent int) {
	if guiManager != nil && guiManager.progressDlg != nil {
		guiManager.progressDlg.Value(percent)
	}
}

func showError(message string) {
	zenity.Error(message, zenity.Title("UrBackup Installer Error"), zenity.Width(400))
}

func showFinalMessage(message string) {
	zenity.Info(message, zenity.Title("UrBackup Installer"), zenity.Width(400))
}

func do_download() error {
	var server_url = unhex("{{ serverurl }}")
	var server_username = unhex("{{ username }}")
	var server_password = unhex("{{ password }}")
	var clientname_prefix = unhex("{{ clientname_prefix }}")
	var group_name = unhex("{{ group_name }}")
	var append_rnd = true
	if "{{ append_rnd }}" == "0" {
		append_rnd = false
	}
	var no_tray = false
	if "{{ notray }}" == "1" {
		no_tray = true
	}
	var silent = false
	if "{{ silent }}" == "1" {
		silent = true
	}
	var linux = false
	if "{{ linux }}" == "1" {
		linux = true
	}

	var server_settings ServerSettings
	server_settings.Url = server_url

	sr, err := get_salt(server_settings, server_username)

	if err != nil {
		return err
	}

	err = login(server_settings, server_username, sr, server_password)

	if err != nil {
		return err
	}

	clientname, err := os.Hostname()

	if err != nil {
		return err
	}

	clientname = clientname_prefix + clientname

	if append_rnd {
		app := make([]byte, 5)
		_, err := rand.Read(app)
		if err != nil {
			panic(err)
		}
		clientname = clientname + "-" + hex.EncodeToString(app)
	}

	setStatus(fmt.Sprintf("Clientname: %s", clientname))

	var installer_name string
	if !linux {
		installer_name = "UrBackup Client Installer.exe"

		if no_tray {
			installer_name = "UrBackupUpdate.exe"
		}
	} else {
		installer_name = "urbackup_client_installer.sh"
	}

	add_client_resp, err := add_client(server_settings, sr, clientname, group_name)

	if err != nil {
		return err
	}

	tmpdir, err := ioutil.TempDir("", "urbackup_installer")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	var file_fn string

	if add_client_resp.Already_exists {
		setStatus("Client already exists")
		status, err := get_status(server_settings, sr)

		if err != nil {
			return err
		}

		if len(status.Client_downloads) == 0 {
			setStatus("Client already exists and login user has probably no right to access existing clients")
			return nil
		}

		for _, client_dl := range status.Client_downloads {
			if client_dl.Name == clientname {

				file, err := download_client(server_settings, sr, client_dl.Id, "", tmpdir, installer_name, linux)

				if err != nil {
					return err
				}

				file_fn = file.Name()
				file.Close()
			}
		}
	} else {
		file, err := download_client(server_settings, sr, add_client_resp.New_clientid, add_client_resp.New_authkey, tmpdir, installer_name, linux)

		if err != nil {
			return err
		}

		file_fn = file.Name()
		file.Close()
	}

	inst_param := ""

	if silent {
		if linux {
			inst_param = " -- silent"
		} else {
			inst_param = "/S"
		}
	}

	if no_tray {
		_ = mod_notray()
	}

	var cmd *exec.Cmd

	if linux {
		cmd = exec.Command("/bin/sh", file_fn, inst_param)
	} else {
		cmd = exec.Command(file_fn, inst_param)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		output := stdoutBuf.String() + stderrBuf.String()
		if len(output) > 4000 {
			output = output[:4000] + "... (truncated)"
		}
		showError(fmt.Sprintf("Failed to start installer. Error: %v\nOutput:\n%s", err, output))
		return err
	}

	return nil
}

func main() {
	var retry = false
	if "{{ retry }}" == "1" {
		retry = true
	}

	// Initialize GUI
	err := initGUI()
	if err != nil {
		// Fallback to error dialog if GUI init fails
		showError(fmt.Sprintf("Failed to initialize GUI: %v", err))
		return
	}
	defer closeGUI()

	var do_retry = true
	for do_retry {
		do_retry = false

		err := do_download()

		if err != nil {
			setStatus(fmt.Sprintf("Error: %s", err.Error()))

			if !retry {
				showFinalMessage(fmt.Sprintf("Installation failed Status %s. Press OK to close.", err.Error()))
			}
		} else {
			// Success message
			setStatus("Installation completed successfully!")
		}

		if retry && err != nil {
			setStatus("Retrying in 30s...")
			do_retry = true
			time.Sleep(30 * time.Second)
		}
	}

}
