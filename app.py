from flask import Flask
from flask import render_template, request
from logging.handlers import RotatingFileHandler
import logging
import flask
import uuid
import threading
import json
import os
import subprocess
import shutil
import time
from io import BytesIO
import binascii

app = Flask(__name__)


if not app.debug:
    file_handler = RotatingFileHandler("/var/log/app/app.log",
                    "a", 100*1024*1024, 10)

    file_handler.setLevel(logging.DEBUG)
    file_handler.setFormatter(logging.Formatter(
                    '%(asctime)s %(levelname)s: %(message)s '
                    '[in %(pathname)s:%(lineno)d]'
                ))

    app.logger.addHandler(file_handler)
    app.logger.setLevel(logging.DEBUG)


@app.route("/")
def home():
    return render_template(
        'index.html',
        title='UrBackup Installer Creator'
    )

@app.route("/create_installer", methods=["POST"])
def create_installer():
    data = request.form["data"]

    app.logger.info("Building installer. Options="+data)

    data = json.loads(data)

    silent="0"

    if "silent" in data and data["silent"]==1:
        silent="1"
	
    sel_os = data["sel_os"] if "sel_os" in data else "win32"
    append_rnd = "1" if "append_rnd" in data and data["append_rnd"]==1 else "0"
    clientname_prefix = data["clientname_prefix"] if "clientname_prefix" in data else ""
    notray = "1" if "notray" in data and data["notray"]==1 else "0"
    linux = "1" if "lin" in sel_os else "0"
    retry = "1" if "retry" in data and data["retry"]==1 else "0"

    installer_go = render_template(
        'main.go',
        serverurl=binascii.hexlify(data["serverurl"].encode()).decode(),
        username=binascii.hexlify(data["username"].encode()).decode(),
        password=binascii.hexlify(data["password"].encode()).decode(),
        silent=silent,
        append_rnd=append_rnd,
        clientname_prefix=binascii.hexlify(clientname_prefix.encode()).decode(),
        notray=notray,
        group_name=binascii.hexlify(data["group_name"].encode()).decode(),
        linux=linux,
        retry=retry
    )

    out_name = "UrBackupClientInstaller.exe"

    if linux == "1":
        out_name = "urbackup_client_installer"

    workdir = uuid.uuid4().hex

    os.mkdir(workdir)

    @flask.after_this_request
    def remove_workdir(response):
        shutil.rmtree(workdir)
        return response

    with open(workdir+"/main.go", "wt") as f:
        f.write(installer_go)

    go_os = "windows"
    go_arch = "386"
    go_arm = "6"

    if sel_os=="win64":
        go_os = "windows"
        go_arch = "amd64"
    elif sel_os == "lin32":
        go_os = "linux"
        go_arch = "386"
    elif sel_os == "lin64":
        go_os = "linux"
        go_arch = "amd64"
    elif sel_os == "linarm32":
        go_os = "linux"
        go_arch = "arm"
        go_arm = "6"
    elif sel_os == "linarm64":
        go_os = "linux"
        go_arch = "arm64"

    env = {"GOARCH": go_arch, "GOOS": go_os, "GOARM": go_arm, "PATH": os.getenv("PATH"), "HOME": os.getenv("HOME")}

    try:
        app.logger.info("run-start")
        output = subprocess.check_output(["go",
            "build", "-o", out_name, '-ldflags=-s -w'], stderr=subprocess.STDOUT, cwd=workdir, env=env)
    except subprocess.CalledProcessError as e:
        app.logger.error("err")
        app.logger.error(e)
        app.logger.error('error>' + e.output.decode()+  '<')
        raise
	
    output = subprocess.check_output(["upx", os.path.join(workdir, out_name)], stderr=subprocess.STDOUT)

    outf = BytesIO()
    with open(os.path.join(workdir, out_name), "rb") as f:
        outf.write(f.read())

    outf.seek(0)

    return flask.send_file(outf, as_attachment=True, attachment_filename=out_name)
